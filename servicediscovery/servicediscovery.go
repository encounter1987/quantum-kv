package servicediscovery

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"os"
)

var CONSUL_HTTP_ADDR = os.Getenv("CONSUL_HTTP_ADDR")

type ConsulClientConfig struct {
	Address string
}

// GetLiveNodes retrieves a list of all live nodes from Consul
func GetLiveNodes() ([]string, error) {
	// Create a new Consul client
	config := api.DefaultConfig()
	config.Address = CONSUL_HTTP_ADDR
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	// Query the health service to get all nodes
	health := client.Health()
	services, _, err := health.Service("quantum-kv", "", true, nil)
	if err != nil {
		return nil, err
	}

	// Extract the list of live nodes
	var liveNodes []string
	for _, service := range services {
		liveNodes = append(liveNodes, service.Node.Address)
	}

	return liveNodes, nil
}

// RegisterNode registers a new node with Consul
func RegisterNode(nodeID, address string) error {
	config := api.DefaultConfig()
	config.Address = CONSUL_HTTP_ADDR
	client, err := api.NewClient(config)
	if err != nil {
		return err
	}

	registration := &api.AgentServiceRegistration{
		ID:      nodeID,
		Name:    "quantum-kv",
		Address: address,
		Port:    11001, // gRPC port
		Check: &api.AgentServiceCheck{
			GRPC:                           fmt.Sprintf("%s:%d", address, 11001),
			Interval:                       "10s", // Check every 10 seconds
			Timeout:                        "5s",  // Timeout after 5 seconds
			DeregisterCriticalServiceAfter: "30s", // Deregister after 1 minute of failure
		},
	}

	return client.Agent().ServiceRegister(registration)
}
