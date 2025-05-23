# Use the official Golang image as a base image
FROM golang:alpine AS builder
# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Install the necessary tools for building the application
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
RUN apk add --no-cache protobuf

# Copy the rest of the application code
COPY . .

# Compile the protobuf files
RUN protoc --go_out=. --go-grpc_out=. -I . server/proto/kv.proto && \
    go build -o quantum-kv main.go

FROM alpine

# Set the working directory inside the container
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/quantum-kv .

# Expose the port the app runs on
EXPOSE 11001