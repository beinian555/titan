package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"titan/internal/worker"
	"titan/pkg/store"
)

func main() {
	// 1. 连接 Etcd
	etcdManager, err := store.NewEtcdManager([]string{"localhost:2379"})
	if err != nil {
		log.Fatalf("Failed to connect to etcd: %v", err)
	}

	// 2. 初始化 Worker Agent
	agent := worker.NewAgent(etcdManager)

	// 3. 启动 Agent
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agent.Run(ctx)

	// 4. 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down worker...")
}
