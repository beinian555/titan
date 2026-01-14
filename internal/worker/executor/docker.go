package executor

import (
	"bytes"
	"context"
	"log"
	"titan/pkg/model"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type DockerExecutor struct {
	cli *client.Client
}

// Init 初始化 Docker 客户端
func NewDockerExecutor() (*DockerExecutor, error) {
	// 自动从环境变量或默认路径连接本地 Docker
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.44"))
	if err != nil {
		return nil, err
	}
	return &DockerExecutor{cli: cli}, nil
}

// Run 真正执行任务的方法
func (e *DockerExecutor) Run(ctx context.Context, job *model.Job) (string, error) {
	log.Printf("🐳 [Docker] Starting job %s...", job.ID)

	// 1. 拉取镜像 (Pull Image)
	// 为了演示快一点，如果本地有镜像可以注释掉这步，或者写个判断
	imageName := "alpine:latest" // 默认用 alpine，体积小
	//if job.Spec.Image != "" {
	//	imageName = job.Spec.Image
	//}

	//log.Printf("   -> Pulling image: %s", imageName)
	//reader, err := e.cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	//if err != nil {
	//	return "", err
	//}
	//io.Copy(os.Stdout, reader) // 把拉取进度打印出来
	//reader.Close()

	// 2. 创建容器 (Create Container)
	resp, err := e.cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		Cmd:   job.Spec.Command, // 例如 ["echo", "hello"]
		Tty:   false,
	}, nil, nil, nil, "")
	if err != nil {
		return "", err
	}

	containerID := resp.ID
	log.Printf("   -> Container created: %s", containerID[:12])

	// 3. 启动容器 (Start Container)
	if err := e.cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}
	log.Printf("   -> Container started, running...")

	// 4. 等待容器结束 (Wait)
	statusCh, errCh := e.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", err
		}
	case <-statusCh:
	}

	// 5. 获取日志 (Logs) - 这是给用户看的
	outReader, err := e.cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", err
	}
	defer outReader.Close()

	// 使用 Buffer 捕获输出
	var buf bytes.Buffer
	// stdcopy 会把 docker 的多路复用流拆分，写入 buf
	// 这里的 output 不再直接打印到 os.Stdout，而是存进内存
	_, err = stdcopy.StdCopy(&buf, &buf, outReader)
	if err != nil {
		return "", err
	}

	log.Printf("✅ [Docker] Job %s finished successfully!", job.ID)

	// 6. 清理容器 (Remove) - 就像 defer 垃圾回收
	e.cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{})

	return buf.String(), nil
}
