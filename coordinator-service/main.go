package main

import (
	"context"
	"log"
	"net"
	"os"

	"github.com/aquare11e/media-dowloader-bot/common/protogen/common"
	coordinatorpb "github.com/aquare11e/media-dowloader-bot/common/protogen/coordinator"
	"github.com/aquare11e/media-dowloader-bot/internal/coordinator"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	servicePort := getEnvOrRaise("SERVICE_PORT")

	transmissionURL := getEnvOrRaise("TRANSMISSION_URL")
	plexURL := getEnvOrRaise("PLEX_URL")
	redisURL := getEnvOrRaise("REDIS_URL")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	pbTypeToDownloadPath := map[common.RequestType]string{
		common.RequestType_FILMS:           getEnvOrRaise("FILMS_DIR_PATH"),
		common.RequestType_SERIES:          getEnvOrRaise("SERIES_DIR_PATH"),
		common.RequestType_CARTOONS:        getEnvOrRaise("CARTOONS_DIR_PATH"),
		common.RequestType_CARTOONS_SERIES: getEnvOrRaise("CARTOONS_SERIES_DIR_PATH"),
		common.RequestType_SHORTS:          getEnvOrRaise("SHORTS_DIR_PATH"),
	}

	// Create Redis client
	redisOptions := &redis.Options{
		Addr: redisURL,
		DB:   0, // use default DB
	}
	if redisPassword != "" {
		redisOptions.Password = redisPassword
	}
	redisClient := redis.NewClient(redisOptions)

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Create gRPC connections to other services
	transmissionConn, err := grpc.NewClient(transmissionURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Transmission service: %v", err)
	}
	defer transmissionConn.Close()

	plexConn, err := grpc.NewClient(plexURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Plex service: %v", err)
	}
	defer plexConn.Close()

	// Create coordinator service
	coordinatorService := coordinator.NewService(transmissionConn, plexConn, redisClient, pbTypeToDownloadPath)

	// Create gRPC server
	grpcServer := grpc.NewServer()
	coordinatorpb.RegisterCoordinatorServiceServer(grpcServer, coordinatorService)
	reflection.Register(grpcServer)

	// Start listening
	lis, err := net.Listen("tcp", ":"+servicePort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Println("Coordinator service is running on port " + servicePort)
	go coordinatorService.StartRecoveryService(ctx)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func getEnvOrRaise(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is not set", key)
	}
	return value
}
