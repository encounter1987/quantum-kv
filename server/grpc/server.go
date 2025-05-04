package server

import (
	"errors"
	"fmt"
	"github.com/encounter1987/quantum-kv/gen/go/kv"
	"github.com/encounter1987/quantum-kv/quantumdb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"os"
)

var ENV string

func init() {
	// read the environment variable ENV
	ENV = os.Getenv("ENV")
}

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
	if err := s.KVStore.Set(req.Key, req.Value); err != nil {

		fmt.Println("error in set value", err)
		if errors.Is(err, quantumdb.ERROR_NOT_A_LEADER) {
			fmt.Println("[SET] not a leader, routing to leader")
			// get the leader address
			// rout the request to the leader
			leaderId, err := s.KVStore.GetLeader()
			if err != nil {
				fmt.Errorf("failed to get leader: %v", err)
				return &kv.SetResponse{Success: false}, err
			}

			fmt.Println("Got the leader:  id", leaderId)

			leaderAddress := getLeaderAddress(leaderId)

			// Use grpc.DialContext instead of grpc.Dial
			conn, err := grpc.NewClient(leaderAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Fatalf("did not connect: %v", err)
			}
			defer conn.Close()

			fmt.Println("connected to leader to route request", leaderAddress)

			defer conn.Close()
			client := kv.NewKeyValueStoreClient(conn)

			// send the request to the leader
			_, err = client.SetValue(ctx, &kv.SetRequest{
				Key:   req.Key,
				Value: req.Value,
			})

			if err != nil {
				fmt.Errorf("failed to set value on leader %s: %v", leaderAddress, err)
				return &kv.SetResponse{Success: false}, err
			}

			return &kv.SetResponse{Success: true}, nil
		}

		return &kv.SetResponse{Success: false}, err
	}

	return &kv.SetResponse{Success: true}, nil
}

func (s *DBServer) GetValue(ctx context.Context, req *kv.GetRequest) (*kv.GetResponse, error) {
	value, found, _ := s.KVStore.Get(req.Key)

	return &kv.GetResponse{Value: value, Found: found}, nil
}

func (s *DBServer) DeleteKey(ctx context.Context, req *kv.DeleteRequest) (*kv.DeleteResponse, error) {
	if err := s.KVStore.Delete(req.Key); err != nil {
		// get the leader address
		// rout the request to the leader
		if errors.Is(err, quantumdb.ERROR_NOT_A_LEADER) {
			fmt.Println("[DELETE] not a leader, routing to leader")
			leaderId, err := s.KVStore.GetLeader()
			if err != nil {
				fmt.Errorf("failed to get leader: %v", err)
				return &kv.DeleteResponse{Success: false}, err
			}

			fmt.Println("Got the leader:  id", leaderId)

			leaderAddress := getLeaderAddress(leaderId)
			conn, err := grpc.Dial(leaderAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				fmt.Errorf("failed to connect to leader %s: %v", leaderAddress, err)
				return &kv.DeleteResponse{Success: false}, err
			}

			fmt.Println("connected to leader to route request to delete", leaderAddress)

			defer conn.Close()
			client := kv.NewKeyValueStoreClient(conn)

			// send the request to the leader
			_, err = client.DeleteKey(ctx, &kv.DeleteRequest{
				Key: req.Key,
			})

			if err != nil {
				fmt.Errorf("failed to delete value on leader %s: %v", leaderAddress, err)
				return &kv.DeleteResponse{Success: false}, err
			}

			fmt.Println("deleted value on leader to route request", leaderAddress)

			return &kv.DeleteResponse{Success: true}, nil
		}

		return &kv.DeleteResponse{Success: false}, err
	}

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
	fmt.Printf("GRPC server is running on %s\n", address)
	return grpcServer.Serve(listener)
}

func getLeaderAddress(leaderId string) string {
	grpcPort := "11001"
	//if ENV == "docker-desktop" {
	//	grpcPort = fmt.Sprintf("%s", map[string]string{
	//		"quantum-kv-node1": "11001",
	//		"quantum-kv-node2": "11002",
	//		"quantum-kv-node3": "11003",
	//	}[leaderId])
	//}
	return fmt.Sprintf("%s:%s", leaderId, grpcPort)
}
