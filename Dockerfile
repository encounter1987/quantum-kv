# Use the official Golang image as a base image
FROM golang:alpine as builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN go build -o quantum-kv main.go

FROM golang:alpine

# Set the working directory inside the container
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/quantum-kv .

# Expose the port the app runs on
EXPOSE 11001