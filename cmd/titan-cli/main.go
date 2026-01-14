package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"titan/pkg/model"
	"titan/pkg/store"
)

func main() {
	// --- 1. 定义命令行参数 ---
	// 任务数量 (默认 1，想压测可以设为 100, 500...)
	taskCount := flag.Int("n", 1, "Number of tasks to submit")
	// 模拟耗时 (默认 1秒，想测长时间任务可以改大)
	sleepTime := flag.Int("t", 1, "Sleep time in seconds for each task")
	// 获取日志 (如果指定了这个 ID，就不提交任务，只查日志)
	jobIDToGet := flag.String("getlog", "", "Get logs for a specific Job ID")

	flag.Parse()

	// --- 2. 连接 Etcd ---
	etcdManager, err := store.NewEtcdManager([]string{"localhost:2379"})
	if err != nil {
		log.Fatalf("❌ Failed to connect to etcd: %v", err)
	}

	// --- 3. 分支 A: 查看日志模式 ---
	if *jobIDToGet != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		logs, err := etcdManager.GetJobLog(ctx, *jobIDToGet)
		if err != nil {
			log.Fatalf("❌ Failed to get logs: %v", err)
		}

		fmt.Printf("\n📄 Logs for Job [%s]:\n", *jobIDToGet)
		fmt.Println("================================================")
		fmt.Println(logs)
		fmt.Println("================================================")
		return // 查完日志直接结束
	}

	// --- 4. 分支 B: 提交任务模式 (支持并发压测) ---
	fmt.Printf("🚀 Starting submission: %d tasks (Simulating %ds work)...\n", *taskCount, *sleepTime)

	var wg sync.WaitGroup
	wg.Add(*taskCount)
	start := time.Now()

	// 并发控制通道 (信号量)，防止一次性并发太高把客户端打挂
	// 限制同时只有 50 个协程在提交任务
	sem := make(chan struct{}, 50)

	for i := 0; i < *taskCount; i++ {
		sem <- struct{}{} // 获取令牌
		go func(id int) {
			defer func() {
				<-sem // 释放令牌
				wg.Done()
			}()

			// 生成唯一 ID
			jobID := fmt.Sprintf("job-%d-%d", time.Now().UnixNano(), id)

			// 构造 Shell 命令：模拟耗时并打印一些信息
			// 例如: "echo Start...; sleep 5; echo Done; ls -l /"
			cmdStr := fmt.Sprintf("echo 'Task %d started on node'; sleep %d; echo 'Task %d finished'; echo 'Here is some file list:'; ls -l /bin | head -n 3", id, *sleepTime, id)

			job := &model.Job{
				ID:   jobID,
				Name: fmt.Sprintf("Job-%d", id),
				Type: model.JobTypeShell,
				Spec: struct {
					Image      string   `json:"image,omitempty"`
					Command    []string `json:"command"`
					Envs       []string `json:"envs"`
					RetryCount int      `json:"retry_count"`
				}{
					Command: []string{"sh", "-c", cmdStr},
				},
				ResReq: model.Resource{
					MilliCPU: 100,       // 0.1 核
					Memory:   1024 * 10, // 10 MB
				},
			}
			job.Status.State = model.JobPending

			// 提交任务
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := etcdManager.CreateJob(ctx, job); err != nil {
				fmt.Printf("❌ Failed to submit job %s: %v\n", jobID, err)
			} else {
				// 如果是单任务，打印详细点；如果是压测，只打印进度
				if *taskCount == 1 {
					fmt.Printf("✅ Job submitted! ID: %s\n", job.ID)
					fmt.Println("💡 View logs later with:")
					fmt.Printf("   go run cmd/titan-cli/main.go -getlog %s\n", job.ID)
				} else if id%50 == 0 {
					fmt.Printf("-> Submitted batch around index %d...\n", id)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	if *taskCount > 1 {
		qps := float64(*taskCount) / duration.Seconds()
		fmt.Printf("\n✅ Stress Test Finished!\n")
		fmt.Printf("   Total Jobs: %d\n", *taskCount)
		fmt.Printf("   Total Time: %v\n", duration)
		fmt.Printf("   Submission QPS: %.2f\n", qps)
	}
}
