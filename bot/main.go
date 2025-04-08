package main

import (
	"log"
	"os"
	"strings"

	coordinator "github.com/aquare11e/media-downloader-bot/common/protogen/coordinator"
	"github.com/aquare11e/media-downloader-bot/internal/bot"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	tokenEnv                 = "TELEGRAM_BOT_TOKEN"
	allowedUsersEnv          = "ALLOWED_USERS"
	coordinatorServiceUrlEnv = "COORDINATOR_SERVICE_URL"
)

func main() {
	token, ok := os.LookupEnv(tokenEnv)
	if !ok {
		log.Fatalf("Environment variable %s is not set", tokenEnv)
	}

	allowedUsersStr, ok := os.LookupEnv(allowedUsersEnv)
	if !ok {
		log.Fatalf("Environment variable %s is not set", allowedUsersEnv)
	}

	coordinatorServiceUrl, ok := os.LookupEnv(coordinatorServiceUrlEnv)
	if !ok {
		log.Fatalf("Environment variable %s is not set", coordinatorServiceUrlEnv)
	}

	// Create gRPC client for coordinator service
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient(coordinatorServiceUrl, opts...)
	if err != nil {
		log.Fatalf("Failed to create gRPC client: %v", err)
	}
	defer conn.Close()

	coordClient := coordinator.NewCoordinatorServiceClient(conn)

	// Create bot with dependencies
	bot, err := bot.NewBot(bot.Config{
		Token:             token,
		AllowedUsers:      strings.Split(allowedUsersStr, ","),
		CoordinatorClient: coordClient,
	})
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	bot.Start()
}
