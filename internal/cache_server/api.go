package cache_server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fanb129/SDCS/proto"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"time"
)

// GET /{key}
//  1. 正常：返回HTTP 200，body为JSON格式的KV结果；
//  2. 错误：返回HTTP 404，body为空。
func (mc *MemoryCache) handelGet(response http.ResponseWriter, request *http.Request) {
	key := request.URL.Path[1:]
	// 判断是否为当前节点处理
	idx := hash(key, len(mc.config.Nodes))
	var v interface{}
	var ok bool
	// 如果 key 存储在当前节点
	if idx == mc.config.Id {
		v, ok = mc.get(key)
	} else {
		// 存储在其他节点，进行rpc调用
		log.Printf("forward to node %d : %s", idx, mc.config.Nodes[idx])
		if mc.grpcCli[idx] == nil {
			mc.startRpcClient()
		}
		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*3)
		defer cancelFunc()
		getRes, err := mc.grpcCli[idx].Get(ctx, &proto.GetRequest{Key: key})
		if err != nil {
			log.Printf("grpc call failed: %v", err)
			ok = false
		} else {
			v, ok = getRes.Value, getRes.Found
		}

	}

	// 正常：返回HTTP 200，body为JSON格式的KV结果
	if ok {
		// 设置 HTTP 状态为 200，并返回 key-value 结果
		response.WriteHeader(http.StatusOK)
		response.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(response).Encode(map[string]interface{}{key: v})
		if err != nil {
			log.Printf("json encode failed: %v", err)
		}
	} else {
		// 如果 key 不存在，返回 404
		response.WriteHeader(http.StatusNotFound)
	}
}

// POST /
// curl -XPOST -H "Content-type: application/json" http://server2/ -d '{"tasks": ["task 1", "task 2", "task 3"]}'
func (mc *MemoryCache) handelSet(response http.ResponseWriter, request *http.Request) {
	// 读取请求体
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(response, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	// 解析 JSON 格式的 key-value 数据
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		http.Error(response, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	for key, value := range data {
		idx := hash(key, len(mc.config.Nodes))
		if idx == mc.config.Id {
			mc.set(key, value)
			break
		}
		log.Printf("forward to node %d : %s", idx, mc.config.Nodes[idx])
		if mc.grpcCli[idx] == nil {
			mc.startRpcClient()
		}
		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*3)
		defer cancelFunc()
		_, err := mc.grpcCli[idx].Set(ctx, &proto.SetRequest{Key: key, Value: value.(string)})
		if err != nil {
			log.Printf("grpc call failed: %v", err)
		}

	}
	response.WriteHeader(http.StatusOK)
}

// DELETE /{key}
func (mc *MemoryCache) handelDelete(response http.ResponseWriter, request *http.Request) {
	key := request.URL.Path[1:]
	idx := hash(key, len(mc.config.Nodes))

	var deleted int
	if idx == mc.config.Id {
		deleted = mc.delete(key)
	} else {
		// 存储在其他节点，进行rpc调用
		log.Printf("forward to node %d : %s", idx, mc.config.Nodes[idx])
		if mc.grpcCli[idx] == nil {
			mc.startRpcClient()
		}
		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*3)
		defer cancelFunc()
		deleteRes, err := mc.grpcCli[idx].Delete(ctx, &proto.DeleteRequest{Key: key})
		if err != nil {
			log.Printf("grpc call failed: %v", err)
			deleted = 0
		} else {
			deleted = int(deleteRes.Deleted)
		}
	}

	fmt.Fprintf(response, "%d", deleted)
	response.WriteHeader(http.StatusOK)
}

func (mc *MemoryCache) handelHttp(response http.ResponseWriter, request *http.Request) {
	log.Printf("request: %s %s\n", request.Method, request.URL)
	switch request.Method {
	case http.MethodGet:
		mc.handelGet(response, request)
	case http.MethodPost:
		mc.handelSet(response, request)
	case http.MethodDelete:
		mc.handelDelete(response, request)
	default:
		http.Error(response, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (mc *MemoryCache) startHttp() error {
	http.HandleFunc("/", mc.handelHttp)
	addr := fmt.Sprintf("0.0.0.0:%d", mc.config.HttpPort)
	log.Printf("http server listening on %s", addr)
	return http.ListenAndServe(addr, nil)
}

// 根据key进行hash，并mod n
func hash(key string, n int) int {
	h := fnv.New32a() // 创建 FNV-1a 32位哈希
	h.Write([]byte(key))
	hashValue := h.Sum32()
	return int(hashValue) % n
}
