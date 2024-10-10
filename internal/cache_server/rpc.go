package cache_server

import (
	"context"
	"errors"
	"github.com/fanb129/SDCS/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
)

func (mc *MemoryCache) Get(ctx context.Context, req *proto.GetRequest) (*proto.GetResponse, error) {
	log.Printf("RPC Get,key=%s\n", req.Key)
	key := req.Key
	value, ok := mc.get(key)
	if !ok {
		return &proto.GetResponse{Found: false}, errors.New("key not found\n")
	}
	return &proto.GetResponse{
		Found: true,
		Key:   key,
		Value: value.(string),
	}, nil
}

func (mc *MemoryCache) Set(ctx context.Context, req *proto.SetRequest) (*proto.SetResponse, error) {
	log.Printf("RPC Set,key=%s,value=%s\n", req.Key, req.Value)
	key, value := req.Key, req.Value
	err := mc.set(key, value)
	if err != nil {
		return &proto.SetResponse{Success: false}, errors.New("set failed\n")
	}
	return &proto.SetResponse{Success: true}, nil
}

func (mc *MemoryCache) Delete(ctx context.Context, req *proto.DeleteRequest) (*proto.DeleteResponse, error) {
	log.Printf("RPC Delete,key=%s\n", req.Key)
	key := req.Key
	deleted := mc.delete(key)
	if deleted == 0 {
		return &proto.DeleteResponse{Deleted: 0}, errors.New("delete failed\n")
	}
	return &proto.DeleteResponse{Deleted: 1}, nil
}

// 注册服务
func (mc *MemoryCache) startRpcServer() error {
	listen, err := net.Listen("tcp", mc.config.Nodes[mc.config.Id])
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	proto.RegisterCacheServiceServer(s, mc)
	log.Printf("RPC server listen on ： %v", listen.Addr())
	err = s.Serve(listen)
	return err
}

// 启动rpc client
func (mc *MemoryCache) startRpcClient() error {
	for i, node := range mc.config.Nodes {
		if i != mc.config.Id {
			conn, err := grpc.NewClient(
				node,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			if err != nil {
				return err
			}
			mc.grpcCli[i] = proto.NewCacheServiceClient(conn)
			log.Printf("grpc client connect to %s", node)
		}
	}
	return nil
}
