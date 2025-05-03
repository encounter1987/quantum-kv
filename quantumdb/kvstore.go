package quantumdb

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

// Node represents a node in the cluster.
type Node struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

// NodeStatus is the Status of a Node returns.
type NodeStatus struct {
	MyId      Node   `json:"my-id"`
	Leader    Node   `json:"leader"`
	Followers []Node `json:"followers"`
}

type KVStore struct {
	RaftDirectory string
	RaftBind      string

	mu sync.Mutex
	kv map[string]string // The key-value store for the system.

	raft *raft.Raft // The consensus mechanism

	logger *log.Logger
}

func NewKVStore(raftDir, raftBind string) *KVStore {
	return &KVStore{
		RaftDirectory: raftDir,
		RaftBind:      raftBind,
		kv:            make(map[string]string),
		logger:        log.New(os.Stderr, "[kvstore] ", log.LstdFlags),
	}
}

func (db *KVStore) Open(enableSingle bool, localID string) error {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(localID)

	addr, err := net.ResolveTCPAddr("tcp", db.RaftBind)
	if err != nil {
		return fmt.Errorf("Failed to resolve raft address: %s", err.Error())
	}

	fmt.Println(db.RaftBind)
	fmt.Println(addr)
	transport, err := raft.NewTCPTransport(db.RaftBind, addr, 3, time.Second, os.Stderr)
	if err != nil {
		return fmt.Errorf("Failed to create raft transport: %s", err.Error())
	}

	snapshotStore, err := raft.NewFileSnapshotStore(db.RaftDirectory, 2, os.Stderr)
	if err != nil {
		return fmt.Errorf("file snapshot store: %s", err)
	}

	var logStore raft.LogStore
	var stableStore raft.StableStore

	boltDB, err := raftboltdb.New(
		raftboltdb.Options{
			Path: filepath.Join(db.RaftDirectory, "raft.db"),
		})
	logStore = boltDB
	stableStore = boltDB

	if err != nil {
		return fmt.Errorf("error initiallizing bolt store: %s", err)
	}

	raftNode, err := raft.NewRaft(config, (*fsm)(db), logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return fmt.Errorf("error creating raft node: %s", err)
	}

	db.raft = raftNode

	if enableSingle {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(localID),
					Address: raft.ServerAddress(db.RaftBind),
				},
			},
		}

		raftNode.BootstrapCluster(configuration)
	}

	return nil
}

func (db *KVStore) AddNode(nodeID, addr string) error {
	db.logger.Printf("adding new node %s with address %s", nodeID, addr)

	configFuture := db.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		db.logger.Printf("failed to get raft configuration: %v", err)
		return err
	}

	for _, server := range configFuture.Configuration().Servers {
		// If a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first.
		if server.ID == raft.ServerID(nodeID) || server.Address == raft.ServerAddress(addr) {
			// if *both* the ID and the address are the same, then join operation is not needed.
			if server.Address == raft.ServerAddress(addr) && server.ID == raft.ServerID(nodeID) {
				db.logger.Printf("node %s at %s already member of cluster, ignoring join request", nodeID, addr)
				return nil
			}

			future := db.raft.RemoveServer(server.ID, 0, 0)
			if err := future.Error(); err != nil {
				return fmt.Errorf("error removing existing node %s at %s: %s", nodeID, addr, err)
			}
		}

	}

	future := db.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if future.Error() != nil {
		return future.Error()
	}
	db.logger.Printf("node %s at %s joined successfully", nodeID, addr)

	return nil
}

func (db *KVStore) Get(key string) (string, bool, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	value, ok := db.kv[key]
	return value, ok, nil
}

func (db *KVStore) Set(key, value string) error {
	if db.raft.State() != raft.Leader {
		return fmt.Errorf("not the leader")
	}

	cmd := &command{
		Type:  PUT,
		Key:   key,
		Value: value,
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("error marshalling command: %s", err)
	}

	future := db.raft.Apply(data, raftTimeout)

	return future.Error()
}

func (db *KVStore) Delete(key string) error {
	if db.raft.State() != raft.Leader {
		return fmt.Errorf("not the leader")
	}

	cmd := &command{
		Type: DELETE,
		Key:  key,
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("error marshalling command: %s", err)
	}

	future := db.raft.Apply(data, raftTimeout)

	return future.Error()
}

func (db *KVStore) Status() (NodeStatus, error) {
	log.Printf("getting status of the cluster")
	leaderServerAddr, leaderId := db.raft.LeaderWithID()
	leader := Node{
		ID:      string(leaderId),
		Address: string(leaderServerAddr),
	}

	servers := db.raft.GetConfiguration().Configuration().Servers
	followers := []Node{}
	myId := Node{
		Address: db.RaftBind,
	}
	for _, server := range servers {
		if server.ID != leaderId {
			followers = append(followers, Node{
				ID:      string(server.ID),
				Address: string(server.Address),
			})
		}

		if string(server.Address) == db.RaftBind {
			myId = Node{
				ID:      string(server.ID),
				Address: string(server.Address),
			}
		}
	}

	return NodeStatus{
		MyId:      myId,
		Leader:    leader,
		Followers: followers,
	}, nil
}
