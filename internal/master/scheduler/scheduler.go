package scheduler

import (
	"context"
	"log"
	"time"

	"titan/pkg/model"
	"titan/pkg/store"
)

// Scheduler 核心调度器结构体
type Scheduler struct {
	store store.Store // 依赖 Store 接口操作 Etcd
}

// NewScheduler 构造函数
func NewScheduler(s store.Store) *Scheduler {
	return &Scheduler{
		store: s,
	}
}

// Run 启动调度主循环 (这是后台常驻 Goroutine)
func (s *Scheduler) Run(ctx context.Context) {
	// 1. Watch 机制：监听 Job 变化 (简历加分项：事件驱动架构)
	jobEventCh := s.store.WatchJobs(ctx)

	log.Println("[Scheduler] Started, watching for new jobs...")

	for {
		select {
		case event := <-jobEventCh:
			// 只处理 Pending (待调度) 的任务
			if event.Job.Status.State == model.JobPending {
				log.Printf("[Scheduler] Detected new job: %s", event.Job.ID)
				// 异步调度，防止阻塞主 Watch 循环
				go s.scheduleOne(ctx, event.Job)
			}
		case <-ctx.Done():
			log.Println("[Scheduler] Stopped.")
			return
		}
	}
}

// scheduleOne 执行单次调度逻辑
func (s *Scheduler) scheduleOne(ctx context.Context, job *model.Job) {
	// Step 1: 获取当前集群所有节点快照
	nodes, err := s.store.ListNodes(ctx)
	if err != nil {
		log.Printf("[Error] Failed to list nodes: %v", err)
		return
	}

	// Step 2: Filter (过滤) - 剔除资源不足的节点
	candidates := s.filterNodes(job, nodes)
	if len(candidates) == 0 {
		log.Printf("[Failed] Job %s pending: no suitable nodes found", job.ID)
		return
	}

	// Step 3: Score (打分) - 选出最优节点 (Bin-packing 策略)
	bestNode := s.scoreNodes(job, candidates)

	// Step 4: Bind (绑定) - 将决策写入 Etcd
	err = s.bind(ctx, job, bestNode.ID)
	if err != nil {
		log.Printf("[Error] Failed to bind job %s to node %s: %v", job.ID, bestNode.ID, err)
	} else {
		log.Printf("[Success] Scheduled Job %s -> Node %s", job.ID, bestNode.ID)
	}
}

// bind 将调度结果持久化
func (s *Scheduler) bind(ctx context.Context, job *model.Job, nodeID string) error {
	job.Status.State = model.JobScheduled
	job.Status.NodeID = nodeID
	job.Status.StartTime = time.Now()

	// 更新 Etcd 中的任务状态
	return s.store.UpdateJob(ctx, job)
}
