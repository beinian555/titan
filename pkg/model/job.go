package model
import "time"
type JobType string

const(
	JobTypeShell JobType = "SHELL"
	JobTypeDocker JobType = "DOCKER"
)

type JobState int

const(
	JobPending   JobState = iota // 等待调度
    JobScheduled                 // 已分配节点，未运行
    JobRunning                   // 正在运行
    JobSuccess                   // 运行成功
    JobFailed                    // 运行失败
    JobCancelled                 // 被取消
)

type Job struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Type        JobType           `json:"type"`
    
    // 任务的具体规格
    Spec struct {
        Image      string   `json:"image,omitempty"` // Docker 镜像 (如: alpine:latest)
        Command    []string `json:"command"`         // 执行命令 (如: ["echo", "hello"])
        Envs       []string `json:"envs"`            // 环境变量
        RetryCount int      `json:"retry_count"`     // 容错机制：最大重试次数
    } `json:"spec"`

    // 资源需求 (Scheduler 根据这个找 Node)
    // 含金量点：声明式资源请求
    ResReq Resource `json:"res_req"`

    // 调度信息
    Status struct {
        State     JobState  `json:"state"`
        NodeID    string    `json:"node_id,omitempty"` // 被分配到了哪个节点
        ExitCode  int       `json:"exit_code"`
        Error     string    `json:"error,omitempty"`
        StartTime time.Time `json:"start_time"`
        EndTime   time.Time `json:"end_time"`
    } `json:"status"`

    // DAG 依赖支持
    // 含金量点：任务编排的核心，必须等 Dependencies 里的 ID 都 Success 才能跑
    Dependencies []string `json:"dependencies"` 
}