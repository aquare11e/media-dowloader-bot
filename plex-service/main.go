package main

import (
	"fmt"
	"log"
	"net"
	"os"

	common "github.com/aquare11e/media-dowloader-bot/common/protogen/common"
	protoPlex "github.com/aquare11e/media-dowloader-bot/common/protogen/plex"
	plexClient "github.com/aquare11e/media-dowloader-bot/internal/plex"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Use an environment variable for the address
	servicePort := os.Getenv("SERVICE_PORT")

	// Check if the address is not set
	if servicePort == "" {
		log.Fatalf("Environment variable SERVICE_PORT is not set")
	}

	plexBaseURL := fmt.Sprintf("http://%s:%s", getEnvOrRaise("PLEX_HOST"), getEnvOrRaise("PLEX_PORT"))
	plexToken := getEnvOrRaise("PLEX_TOKEN")
	pbTypeToCategoryId := map[common.RequestType]string{
		common.RequestType_FILMS:           getEnvOrRaise("PLEX_CATEGORY_FILMS"),
		common.RequestType_SERIES:          getEnvOrRaise("PLEX_CATEGORY_SERIES"),
		common.RequestType_CARTOONS:        getEnvOrRaise("PLEX_CATEGORY_CARTOONS"),
		common.RequestType_CARTOONS_SERIES: getEnvOrRaise("PLEX_CATEGORY_CARTOONS_SERIES"),
		common.RequestType_SHORTS:          getEnvOrRaise("PLEX_CATEGORY_SHORTS"),
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", servicePort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server := grpc.NewServer()
	protoPlex.RegisterPlexServiceServer(server, plexClient.NewPlexService(plexBaseURL, plexToken, pbTypeToCategoryId))
	reflection.Register(server)

	log.Printf("server listening at %v", listener.Addr())
	if err = server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func getEnvOrRaise(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is not set", key)
	}
	return value
}
