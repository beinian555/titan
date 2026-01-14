package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"titan/internal/master/scheduler"
	"titan/pkg/store"
)

func main() {
	// 1. 初始化 Etcd 连接
	// 假设我们在本地跑 Etcd，端口通常是 2379
	etcdManager, err := store.NewEtcdManager([]string{"localhost:2379"})
	if err != nil {
		log.Fatalf("Failed to connect to etcd: %v", err)
	}
	log.Println("Connected to Etcd successfully.")

	// 2. 初始化调度器 (依赖注入)
	sched := scheduler.NewScheduler(etcdManager)

	// 3. 启动调度器 (后台运行)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sched.Run(ctx)

	// 4. (未来) 这里还要启动 API Server (HTTP/gRPC) 接收用户请求
	// go apiServer.Run()

	// 5. 优雅退出 (Graceful Shutdown)
	// 等待 Ctrl+C 信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down master...")
}
