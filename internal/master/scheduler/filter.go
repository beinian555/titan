package scheduler

import (
	"log"
	"titan/pkg/model"
)

// filterNodes 遍历节点，返回满足硬性条件的候选者
func (s *Scheduler) filterNodes(job *model.Job, nodes []*model.Node) []*model.Node {
	candidates := make([]*model.Node, 0)

	for _, node := range nodes {
		if s.checkNode(job, node) {
			candidates = append(candidates, node)
		}
	}
	return candidates
}

// checkNode 执行具体的 Predicate 检查逻辑
func (s *Scheduler) checkNode(job *model.Job, node *model.Node) bool {
	// 1. 检查节点健康状态
	if node.Status != model.NodeReady {
		return false
	}

	// 2. 资源检查 (CPU & Memory)
	// 计算剩余资源 = 总容量 - 已分配
	freeCpu := node.TotalCap.MilliCPU - node.Allocated.MilliCPU
	freeMem := node.TotalCap.Memory - node.Allocated.Memory

	// 这里的 ResReq 是我们在 pkg/model/job.go 里定义的
	if freeCpu < job.ResReq.MilliCPU {
		log.Printf("[Filter] Node %s filtered: Insufficient CPU (Free: %d, Need: %d)",
			node.ID, freeCpu, job.ResReq.MilliCPU)
		return false
	}

	if freeMem < job.ResReq.Memory {
		log.Printf("[Filter] Node %s filtered: Insufficient Memory (Free: %d, Need: %d)",
			node.ID, freeMem, job.ResReq.Memory)
		return false
	}

	return true
}
