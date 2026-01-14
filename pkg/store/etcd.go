package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"titan/pkg/model"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// 定义 Key 的前缀 (Schema Design)
const (
	JobKeyPrefix  = "/titan/jobs/"
	NodeKeyPrefix = "/titan/nodes/"
)

type EtcdManager struct {
	client *clientv3.Client
}

// NewEtcdManager 初始化 Etcd 连接
func NewEtcdManager(endpoints []string) (*EtcdManager, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &EtcdManager{client: cli}, nil
}

// ---------------------------------------------------------
// Job 相关实现
// ---------------------------------------------------------

func (e *EtcdManager) CreateJob(ctx context.Context, job *model.Job) error {
	key := JobKeyPrefix + job.ID
	return e.putValue(ctx, key, job)
}

func (e *EtcdManager) GetJob(ctx context.Context, id string) (*model.Job, error) {
	// ... (代码略，逻辑类似 ListNodes，为了节省篇幅只写核心的)
	return nil, nil
}

func (e *EtcdManager) UpdateJob(ctx context.Context, job *model.Job) error {
	key := JobKeyPrefix + job.ID
	return e.putValue(ctx, key, job)
}

// WatchJobs 核心难点：将 Etcd 的 Watch 转换为业务 Channel
func (e *EtcdManager) WatchJobs(ctx context.Context) <-chan JobEvent {
	eventChan := make(chan JobEvent)

	// 启动一个协程在后台一直监听
	go func() {
		// 监听 /titan/jobs/ 前缀下的所有变化
		watchChan := e.client.Watch(ctx, JobKeyPrefix, clientv3.WithPrefix())

		for watchResp := range watchChan {
			for _, ev := range watchResp.Events {
				var eventType JobEventType
				switch ev.Type {
				case clientv3.EventTypePut:
					eventType = JobUpdate // 这里的 Create 和 Update 在 Etcd 都是 Put
				case clientv3.EventTypeDelete:
					eventType = JobDelete
				}

				// 反序列化 Job 数据
				var job model.Job
				if err := json.Unmarshal(ev.Kv.Value, &job); err != nil {
					log.Printf("[Etcd] Failed to unmarshal job: %v", err)
					continue
				}

				// 发送给调度器
				eventChan <- JobEvent{
					Type: eventType,
					Job:  &job,
				}
			}
		}
		close(eventChan)
	}()

	return eventChan
}

// ---------------------------------------------------------
// Node 相关实现
// ---------------------------------------------------------

func (e *EtcdManager) RegisterNode(ctx context.Context, node *model.Node) error {
	key := NodeKeyPrefix + node.ID
	// 这里通常需要加上 Lease (租约)，为了简化先不写
	return e.putValue(ctx, key, node)
}

func (e *EtcdManager) ListNodes(ctx context.Context) ([]*model.Node, error) {
	// 获取 /titan/nodes/ 下的所有 Key
	resp, err := e.client.Get(ctx, NodeKeyPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	nodes := make([]*model.Node, 0)
	for _, kv := range resp.Kvs {
		var node model.Node
		if err := json.Unmarshal(kv.Value, &node); err != nil {
			log.Printf("Failed to unmarshal node: %v", err)
			continue
		}
		nodes = append(nodes, &node)
	}
	return nodes, nil
}

// ---------------------------------------------------------
// 辅助方法 (Helpers)
// ---------------------------------------------------------

// putValue 封装通用的 JSON 序列化 + Put 操作
func (e *EtcdManager) putValue(ctx context.Context, key string, val interface{}) error {
	bytes, err := json.Marshal(val)
	if err != nil {
		return err
	}
	_, err = e.client.Put(ctx, key, string(bytes))
	return err
}

// ---------------------------------------------------------
// Log 相关实现
// ---------------------------------------------------------

func (e *EtcdManager) SaveJobLog(ctx context.Context, jobID string, logs string) error {
	key := "/titan/logs/" + jobID
	// 直接存字符串，或者像之前一样 json 包装一下也可以
	// 这里为了简单，我们构造一个简单的结构体存进去，或者直接存字符串
	// 为了复用 putValue，我们把日志包成一个对象
	data := map[string]string{
		"job_id":  jobID,
		"content": logs,
	}
	return e.putValue(ctx, key, data)
}

func (e *EtcdManager) GetJobLog(ctx context.Context, jobID string) (string, error) {
	key := "/titan/logs/" + jobID
	resp, err := e.client.Get(ctx, key)
	if err != nil {
		return "", err
	}
	if len(resp.Kvs) == 0 {
		return "", fmt.Errorf("log not found for job %s", jobID)
	}

	// 反序列化
	var data map[string]string
	if err := json.Unmarshal(resp.Kvs[0].Value, &data); err != nil {
		return "", err
	}
	return data["content"], nil
}
