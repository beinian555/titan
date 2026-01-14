package store

import (
	"context"
	"titan/pkg/model"
)

// JobEventType 定义监听事件类型
type JobEventType int

const (
	JobCreate JobEventType = iota
	JobUpdate
	JobDelete
)

// JobEvent 包装了 Etcd 中发生的事件
// 调度器通过这个结构体知道有新任务来了
type JobEvent struct {
	Type JobEventType
	Job  *model.Job
}

// Store 接口定义了系统对存储层的所有需求
// 任何实现了这个接口的 Struct (比如 EtcdManager) 都可以被注入到调度器中
type Store interface {
	// --- Job 相关 ---

	// CreateJob 提交新任务
	CreateJob(ctx context.Context, job *model.Job) error

	// GetJob 获取单个任务详情
	GetJob(ctx context.Context, id string) (*model.Job, error)

	// UpdateJob 更新任务状态 (调度器 Bind 时调用)
	UpdateJob(ctx context.Context, job *model.Job) error

	SaveJobLog(ctx context.Context, jobID string, logs string) error
	GetJobLog(ctx context.Context, jobID string) (string, error)
	// WatchJobs 监听任务变化 (返回一个只读通道)
	WatchJobs(ctx context.Context) <-chan JobEvent

	// --- Node 相关 ---

	// RegisterNode 节点注册 (Worker 启动时调用)
	RegisterNode(ctx context.Context, node *model.Node) error

	// ListNodes 获取所有节点 (调度器 Filter 时调用)
	ListNodes(ctx context.Context) ([]*model.Node, error)
}
