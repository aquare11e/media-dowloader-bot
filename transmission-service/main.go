package main

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"os"

	transmissionpb "github.com/aquare11e/media-dowloader-bot/common/protogen/transmission"
	"github.com/aquare11e/media-dowloader-bot/internal/transmission"
	transmissionrpc "github.com/hekmon/transmissionrpc/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Get environment variables
	transmissionHost := getEnvOrDefault("TRANSMISSION_HOST", "")
	transmissionPort := getEnvOrDefault("TRANSMISSION_PORT", "")
	transmissionUser := getEnvOrDefault("TRANSMISSION_USER", "")
	transmissionPassword := getEnvOrDefault("TRANSMISSION_PASSWORD", "")
	servicePort := getEnvOrDefault("SERVICE_PORT", "50052")

	// Create the endpoint URL
	endpoint, err := url.Parse(fmt.Sprintf("http://%s:%s@%s:%s/transmission/rpc", transmissionUser, transmissionPassword, transmissionHost, transmissionPort))
	if err != nil {
		log.Fatalf("Failed to parse endpoint URL: %v", err)
	}

	// Initialize Transmission client
	client, err := transmissionrpc.New(endpoint, nil)
	if err != nil {
		log.Fatalf("Failed to create Transmission client: %v", err)
	}

	// Create gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", servicePort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	transmissionpb.RegisterTransmissionServiceServer(s, transmission.NewServer(client))
	reflection.Register(s)

	log.Printf("Server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
