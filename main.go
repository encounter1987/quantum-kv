package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/encounter1987/quantum-kv/gen/go/kv"
	"github.com/encounter1987/quantum-kv/quantumdb"
	grpcServer "github.com/encounter1987/quantum-kv/server/grpc"
	"github.com/encounter1987/quantum-kv/servicediscovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"os"
	"os/signal"
)

// Command line defaults
const (
	DefaultGRPCAddr = "localhost:11001"
	DefaultRaftAddr = "localhost:12001"
)

// Command line parameters
var grpcAddr string
var raftAddr string
var nodeID string

func init() {
	flag.StringVar(&grpcAddr, "grpcAddr", DefaultGRPCAddr, "Set the GRPC bind address")
	flag.StringVar(&raftAddr, "raddr", DefaultRaftAddr, "Set Raft bind address")
	flag.StringVar(&nodeID, "id", "", "Node ID. If not set, same as Raft bind address")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <raft-data-path> \n", os.Args[0])
		flag.PrintDefaults()
	}

}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "No Raft storage directory specified\n")
		os.Exit(1)
	}

	if nodeID == "" {
		nodeID = raftAddr
	}

	// Ensure Raft storage exists.
	raftDir := flag.Arg(0)
	if raftDir == "" {
		log.Fatalln("No Raft storage directory specified")
	}
	if err := os.MkdirAll(raftDir, 0700); err != nil {
		log.Fatalf("failed to create path for Raft storage: %s", err.Error())
	}

	kvstore := quantumdb.NewKVStore(raftDir, raftAddr)

	liveRaftNodes, err := servicediscovery.GetLiveNodes()
	if err != nil {
		fmt.Errorf("failed to get live nodes from Consul: %w", err)
		return
	}

	fmt.Println("Found live nodes ", liveRaftNodes)

	if err := kvstore.Open(len(liveRaftNodes) == 0, nodeID); err != nil {
		log.Fatalf("failed to open KV store: %s", err.Error())
	}

	server := grpcServer.New(kvstore)
	if err := server.StartGRPCServer(grpcAddr); err != nil {
		log.Fatalf("failed to start gRPC server: %s", err.Error())
	}

	// Join to existing Raft cluster
	if err := joinRaftCluster(raftAddr, grpcAddr, liveRaftNodes); err != nil {
		log.Fatalf("failed to join Raft cluster: %s", err.Error())
	}

	// Register the node with Consul
	servicediscovery.RegisterNode(nodeID, grpcAddr)

	// quantum-kv is up and running!
	log.Printf("quantum-kv started successfully, listening on http://%s", grpcAddr)

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	log.Println("quantum-kv exiting")

}

func joinRaftCluster(raftAddr, grpcAddr string, liveRaftNodes []string) error {

	for _, liveRaftNode := range liveRaftNodes {
		if liveRaftNode == raftAddr {
			continue
		}
		err := join(liveRaftNode, grpcAddr, nodeID)
		if err != nil {
			return fmt.Errorf("failed to join Raft cluster: %w", err)
		} else {
			log.Printf("Joined Raft cluster at %s", liveRaftNode)
			return nil
		}
	}

	log.Printf("No other nodes found in the cluster, starting the first one")

	return nil
}

func join(joinAddr, raftAddr, nodeID string) error {
	conn, err := grpc.Dial(joinAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to join address %s: %w", joinAddr, err)
	}
	defer conn.Close()

	client := kv.NewKeyValueStoreClient(conn)

	_, err = client.AddNode(context.Background(), &kv.AddNodeRequest{
		NodeId:  nodeID,
		Address: raftAddr,
	})
	if err != nil {
		return fmt.Errorf("failed to add node to cluster: %w", err)
	}

	return nil
}
