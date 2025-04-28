package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/encounter1987/quantum-kv/gen/go/kv"
	"github.com/encounter1987/quantum-kv/quantumdb"
	grpcServer "github.com/encounter1987/quantum-kv/server/grpc"
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
var joinAddr string
var nodeID string

func init() {
	flag.StringVar(&grpcAddr, "grpcAddr", DefaultGRPCAddr, "Set the GRPC bind address")
	flag.StringVar(&raftAddr, "raddr", DefaultRaftAddr, "Set Raft bind address")
	flag.StringVar(&joinAddr, "join", "", "Set join address, if any")
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
	if err := kvstore.Open(joinAddr == "", nodeID); err != nil {
		log.Fatalf("failed to open KV store: %s", err.Error())
	}

	server := grpcServer.New(kvstore)
	if err := server.StartGRPCServer(grpcAddr); err != nil {
		log.Fatalf("failed to start gRPC server: %s", err.Error())
	}

	// If join was specified, make the join request.
	if joinAddr != "" {
		if err := join(joinAddr, raftAddr, nodeID); err != nil {
			log.Fatalf("failed to join node at %s: %s", joinAddr, err.Error())
		}
	}

	// quantum-kv is up and running!
	log.Printf("quantum-kv started successfully, listening on http://%s", grpcAddr)

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	log.Println("quantum-kv exiting")

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
