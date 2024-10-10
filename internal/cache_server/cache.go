package cache_server

import (
	"github.com/fanb129/SDCS/proto"
	"log"
	"sync"
)

type CacheServer interface {
	Start() error
	Stop() error
}

type MemoryCache struct {
	mu                                    sync.RWMutex // 读写锁
	cache                                 map[string]interface{}
	proto.UnimplementedCacheServiceServer // 实现grpc
	config                                MemoryCacheConfig
	grpcCli                               []proto.CacheServiceClient // grpc客户端
}

type MemoryCacheConfig struct {
	HttpPort int      // http 端口
	Nodes    []string // 集群rpc地址
	Id       int      // rpc地址所在索引
}

func NewMemoryCache(config MemoryCacheConfig) *MemoryCache {
	return &MemoryCache{
		cache:   make(map[string]interface{}),
		config:  config,
		grpcCli: make([]proto.CacheServiceClient, len(config.Nodes)),
	}
}

func (mc *MemoryCache) Start() error {
	// start http server
	go func() {
		if err := mc.startHttp(); err != nil {
			log.Fatalln("start http error:", err)
		}
	}()

	// start grpc server
	go func() {
		if err := mc.startRpcServer(); err != nil {
			log.Fatalln("start rpc server err : ", err)
		}
	}()

	// start grpc client
	go func() {
		if err := mc.startRpcClient(); err != nil {
			log.Fatalln("start rpc client error : ", err)
		}
	}()

	return nil
}

func (mc *MemoryCache) Stop() error {
	return nil
}

func (mc *MemoryCache) set(key string, value interface{}) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.cache[key] = value
	return nil
}

func (mc *MemoryCache) get(key string) (interface{}, bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	value, exits := mc.cache[key]
	return value, exits
}

func (mc *MemoryCache) delete(key string) int {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if _, exits := mc.cache[key]; exits {
		delete(mc.cache, key)
		return 1
	}
	return 0
}
