package main

import (
	"context"
	"log"
	"time"

	pb "github.com/encounter1987/quantum-kv/gen/go/kv"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewKeyValueStoreClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Set a value
	_, err = c.SetValue(ctx, &pb.SetRequest{Key: "foo", Value: "bar"})
	if err != nil {
		log.Fatalf("Could not set value: %v", err)
	}
	log.Println("Set value")

	// Get a value
	res, err := c.GetValue(ctx, &pb.GetRequest{Key: "foo"})
	if err != nil {
		log.Fatalf("Could not get value: %v", err)
	}

	if res.Found {
		log.Printf("Found value: %s", res.Value)
	} else {
		log.Println("Value not found")
	}
}
