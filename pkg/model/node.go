package model

// NodeStatus 节点健康状态
type NodeStatus string

const (
    NodeReady   NodeStatus = "READY"
    NodeOffline NodeStatus = "OFFLINE" // 心跳超时
)

type Node struct {
    ID      string     `json:"id"`       // 唯一标识，通常是 UUID 或 Hostname
    IP      string     `json:"ip"`       // Worker 的 IP 地址，用于 gRPC 通信
    Version string     `json:"version"`  // Worker 版本号
    
    // 资源视图
    // Total: 物理机总资源
    // Allocated: 已经被任务占用的资源
    // 含金量点：Master 调度时只需计算 Total - Allocated
    TotalCap  Resource `json:"total_cap"`
    Allocated Resource `json:"allocated"`

    Status         NodeStatus `json:"status"`
    LastHeartbeat  int64      `json:"last_heartbeat"` // Unix 时间戳
}