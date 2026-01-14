package worker

import (
	"context"
	"log"
	"os"
	"time"

	"titan/internal/worker/executor"
	"titan/pkg/model"
	"titan/pkg/store"
)

type Agent struct {
	ID       string
	store    store.Store
	executor *executor.DockerExecutor
}

func NewAgent(s store.Store) *Agent {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "worker-node-01"
	}

	// 初始化 Docker 执行器
	exec, err := executor.NewDockerExecutor()
	if err != nil {
		log.Fatalf("Failed to init docker executor: %v", err)
	}

	return &Agent{
		ID:       hostname,
		store:    s,
		executor: exec,
	}
}

func (a *Agent) Run(ctx context.Context) {
	// 1. 启动心跳
	go a.startHeartbeat(ctx)

	// 2. 启动任务监听
	log.Printf("[Worker] Waiting for jobs assigned to %s...", a.ID)
	a.watchJobs(ctx)
}

func (a *Agent) startHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	a.register(ctx)
	for {
		select {
		case <-ticker.C:
			a.register(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (a *Agent) watchJobs(ctx context.Context) {
	eventCh := a.store.WatchJobs(ctx)

	for event := range eventCh {
		job := event.Job
		// 只有当任务被更新，且分配给我，且状态是 Scheduled 时，才处理
		if job.Status.NodeID == a.ID && job.Status.State == model.JobScheduled {
			log.Printf("[Worker] ⚡ Received job: %s", job.ID)
			// 异步执行
			go a.executeJob(ctx, job)
		}
	}
}

// executeJob 执行任务并更新状态 (关键修改在这里！)
func (a *Agent) executeJob(ctx context.Context, job *model.Job) {
	// 1. 更新状态为 Running
	job.Status.State = model.JobRunning
	a.store.UpdateJob(ctx, job)

	// 2. 调用 Docker 执行 (接收两个返回值：output 和 err)
	output, err := a.executor.Run(ctx, job)

	// 3. 根据结果更新最终状态
	if err != nil {
		log.Printf("Job failed: %v", err)
		job.Status.State = model.JobFailed
		job.Status.Error = err.Error()
	} else {
		job.Status.State = model.JobSuccess
	}

	job.Status.EndTime = time.Now()
	a.store.UpdateJob(ctx, job)

	// 4. 上传日志 (不管成功失败，只要有日志就上传)
	if output != "" {
		err := a.store.SaveJobLog(ctx, job.ID, output)
		if err != nil {
			log.Printf("Failed to save job log: %v", err)
		} else {
			log.Printf("📝 Logs saved to Etcd for job %s", job.ID)
		}
	}
}

func (a *Agent) register(ctx context.Context) {
	// 简单上报节点信息
	node := &model.Node{
		ID:      a.ID,
		IP:      "127.0.0.1",
		Version: "v1.0",
		Status:  model.NodeReady,
		// 这里恢复成真实的资源 (或者你之前修改过的 Mock 数据)
		TotalCap: model.Resource{
			MilliCPU: 4000,
			Memory:   1024 * 1024 * 1024 * 8,
		},
		LastHeartbeat: time.Now().Unix(),
	}
	a.store.RegisterNode(ctx, node)
}
