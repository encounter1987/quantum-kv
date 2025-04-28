package main

import (
	"flag"
	"fmt"
	"github.com/encounter1987/quantum-kv/quantumdb"
	grpcServer "github.com/encounter1987/quantum-kv/server/grpc"
	"log"
	"os"
)

// Command line defaults
const (
	DefaultGRPCAddr = "localhost:1001"
	DefaultRaftAddr = "localhost:2001"
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

}
