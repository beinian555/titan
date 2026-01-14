package scheduler

import "titan/pkg/model"

// scoreNodes 给候选节点打分并返回最高分节点
func (s *Scheduler) scoreNodes(job *model.Job, nodes []*model.Node) *model.Node {
	var bestNode *model.Node
	maxScore := -1

	for _, node := range nodes {
		score := s.calculateScore(job, node)

		// 贪心选择：只要分数更高就替换
		if score > maxScore {
			maxScore = score
			bestNode = node
		}
	}
	return bestNode
}

// calculateScore 计算单个节点的得分 (0-100)
// 策略：Bin-packing (堆叠) -> 资源利用率越高，得分越高
// 目的：将任务集中到少数节点，留出大块空闲资源给未来的大任务
func (s *Scheduler) calculateScore(job *model.Job, node *model.Node) int {
	// 预测分配后的资源使用量
	newCpuUsed := node.Allocated.MilliCPU + job.ResReq.MilliCPU
	newMemUsed := node.Allocated.Memory + job.ResReq.Memory

	// 计算 CPU 分数 (使用率百分比 * 10)
	// 例如：总共 1000m，用掉了 800m，得分 = (800/1000)*10 = 8分
	cpuScore := 0
	if node.TotalCap.MilliCPU > 0 {
		cpuScore = int((float64(newCpuUsed) / float64(node.TotalCap.MilliCPU)) * 10)
	}

	// 计算 Memory 分数
	memScore := 0
	if node.TotalCap.Memory > 0 {
		memScore = int((float64(newMemUsed) / float64(node.TotalCap.Memory)) * 10)
	}

	// 简单加权求和
	return cpuScore + memScore
}
