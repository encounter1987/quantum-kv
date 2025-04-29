package server

import (
	"fmt"
	"github.com/encounter1987/quantum-kv/gen/go/kv"
	"github.com/encounter1987/quantum-kv/quantumdb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
)

type DBServer struct {
	kv.KeyValueStoreServer
	KVStore *quantumdb.KVStore
}

func New(store *quantumdb.KVStore) *DBServer {
	return &DBServer{
		KVStore: store,
	}
}

func (s *DBServer) SetValue(ctx context.Context, req *kv.SetRequest) (*kv.SetResponse, error) {
	s.KVStore.Set(req.Key, req.Value)

	return &kv.SetResponse{Success: true}, nil
}

func (s *DBServer) GetValue(ctx context.Context, req *kv.GetRequest) (*kv.GetResponse, error) {
	value, found, _ := s.KVStore.Get(req.Key)

	return &kv.GetResponse{Value: value, Found: found}, nil
}

func (s *DBServer) DeleteValue(ctx context.Context, req *kv.DeleteRequest) (*kv.DeleteResponse, error) {
	s.KVStore.Delete(req.Key)

	return &kv.DeleteResponse{Success: true}, nil
}

func (s *DBServer) AddNode(ctx context.Context, req *kv.AddNodeRequest) (*kv.AddNodeResponse, error) {
	err := s.KVStore.AddNode(req.NodeId, req.Address)
	if err != nil {
		return nil, err
	}

	return &kv.AddNodeResponse{Success: true}, nil
}

func (s *DBServer) Status(ctx context.Context, req *kv.StatusRequest) (*kv.StatusResponse, error) {
	status, err := s.KVStore.Status()
	if err != nil {
		return nil, err
	}

	followers := make([]*kv.Node, 0, len(status.Followers))
	for _, follower := range status.Followers {
		followers = append(followers, &kv.Node{
			Id:      follower.ID,
			Address: follower.Address,
		})
	}

	response := &kv.StatusResponse{
		MyId: &kv.Node{
			Id:      status.MyId.ID,
			Address: status.MyId.Address,
		},
		Leader: &kv.Node{
			Id:      status.Leader.ID,
			Address: status.Leader.Address,
		},
		Followers: followers,
	}

	return response, nil
}

func (s *DBServer) StartGRPCServer(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}
	grpcServer := grpc.NewServer()
	kv.RegisterKeyValueStoreServer(grpcServer, s)
	fmt.Printf("gRPC server is running on %s\n", address)
	return grpcServer.Serve(listener)
}
