package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	coordinator "github.com/aquare11e/media-downloader-bot/common/protogen/coordinator"
	"github.com/aquare11e/media-downloader-bot/internal/bot"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	tokenEnv                 = "TELEGRAM_BOT_TOKEN"
	allowedUserIdsEnv        = "ALLOWED_USER_IDS"
	coordinatorServiceUrlEnv = "COORDINATOR_SERVICE_URL"
	redisUrlEnv              = "REDIS_URL"
	redisPasswordEnv         = "REDIS_PASSWORD"
)

func main() {
	token, ok := os.LookupEnv(tokenEnv)
	if !ok {
		log.Fatalf("Environment variable %s is not set", tokenEnv)
	}

	allowedUsersStr, ok := os.LookupEnv(allowedUserIdsEnv)
	if !ok {
		log.Fatalf("Environment variable %s is not set", allowedUserIdsEnv)
	}

	coordinatorServiceUrl, ok := os.LookupEnv(coordinatorServiceUrlEnv)
	if !ok {
		log.Fatalf("Environment variable %s is not set", coordinatorServiceUrlEnv)
	}

	redisUrl, ok := os.LookupEnv(redisUrlEnv)
	if !ok {
		log.Fatalf("Environment variable %s is not set", redisUrlEnv)
	}

	redisOptions := &redis.Options{
		Addr: redisUrl,
		DB:   0,
	}

	redisPassword, okRedisPassword := os.LookupEnv(redisPasswordEnv)
	if okRedisPassword {
		redisOptions.Password = redisPassword
	}

	redisClient := redis.NewClient(redisOptions)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to ping Redis: %v", err)
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

	allowedUserIds := make([]int64, 0)
	for _, s := range strings.Split(allowedUsersStr, ",") {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			log.Fatalf("Failed to parse allowed user ID: %v", err)
		}
		allowedUserIds = append(allowedUserIds, id)
	}

	// Create bot with dependencies
	bot, err := bot.NewBot(token, allowedUserIds, coordClient, redisClient)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	bot.Start()
}
