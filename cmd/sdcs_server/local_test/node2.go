package main

import (
	"flag"
	"github.com/fanb129/SDCS/internal/cache_server"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	// 定义flag参数
	httpPort := flag.String("http_port", "9529", "HTTP Port")
	nodesStr := flag.String("nodes", "127.0.0.1:8887,127.0.0.1:8888,127.0.0.1:8889", "gRPC Nodes, separated by commas")
	nodeID := flag.Int("node_id", 2, "Current Node Id")

	// 解析参数
	flag.Parse()

	port, err := strconv.Atoi(*httpPort)
	if err != nil {
		panic(err)
	}
	// 把Nodes字符串转为切片
	nodes := strings.Split(*nodesStr, ",")

	var cacheServer cache_server.CacheServer

	config := cache_server.MemoryCacheConfig{
		HttpPort: port,
		Nodes:    nodes,
		Id:       *nodeID,
	}

	cacheServer = cache_server.NewMemoryCache(config)

	if err := cacheServer.Start(); err != nil {
		panic(err)
	}

	// 接受终止信号
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if err := cacheServer.Stop(); err != nil {
		panic(err)
	}
}
