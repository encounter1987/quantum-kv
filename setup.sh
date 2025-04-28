protoc --go_out=. --go-grpc_out=. -I . server/proto/kv.proto

go build -o exec-quantum-kv main.go