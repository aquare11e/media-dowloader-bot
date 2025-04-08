# Plex Service

A gRPC service for managing Plex media server libraries.

## Overview

This service provides functionality to scan and update Plex libraries through a gRPC interface. It supports different types of media categories including films, series, cartoons, cartoon series, and shorts.

## Configuration

The service requires the following environment variables to be set:

- `SERVICE_PORT`: The port number on which the gRPC server will listen.
- `PLEX_HOST`: The hostname or IP address of your Plex server.
- `PLEX_PORT`: The port number on which your Plex server is running.
- `PLEX_TOKEN`: Your Plex authentication token.
- `PLEX_CATEGORY_FILMS`: The library ID for films in your Plex server.
- `PLEX_CATEGORY_SERIES`: The library ID for series in your Plex server.
- `PLEX_CATEGORY_CARTOONS`: The library ID for cartoons in your Plex server.
- `PLEX_CATEGORY_CARTOONS_SERIES`: The library ID for cartoon series in your Plex server.
- `PLEX_CATEGORY_SHORTS`: The library ID for shorts in your Plex server.

## Building and Running

1. Set up environment variables:
```bash
export SERVICE_PORT=50051
export PLEX_HOST=your-plex-server
export PLEX_PORT=32400
export PLEX_TOKEN=your-plex-token
export PLEX_CATEGORY_FILMS=1
export PLEX_CATEGORY_SERIES=2
export PLEX_CATEGORY_CARTOONS=3
export PLEX_CATEGORY_CARTOONS_SERIES=4
export PLEX_CATEGORY_SHORTS=5
```

2. Build and run the service:
```bash
go build -o plex-service
./plex-service
```

## API Documentation

### UpdateCategory

Updates a specific Plex library category.

#### Request
```protobuf
message UpdateCategoryRequest {
  RequestType type = 1;
}

enum RequestType {
  FILMS = 0;
  SERIES = 1;
  CARTOONS = 2;
  CARTOONS_SERIES = 3;
  SHORTS = 4;
}
```

#### Response
```protobuf
message UpdateCategoryResponse {
}
```

#### Example Usage
```go
client := NewPlexServiceClient(conn)
response, err := client.UpdateCategory(ctx, &UpdateCategoryRequest{
    Type: protoPlex.RequestType_FILMS,
})
```

## Testing with gRPCurl

You can use `grpcurl` to test the service:

```bash
grpcurl -plaintext -d '{"type": 0}' localhost:50051 protoPlex.PlexService/UpdateCategory
```

Where `type` values are:
- `0` for FILMS
- `1` for SERIES
- `2` for CARTOONS
- `3` for CARTOONS_SERIES
- `4` for SHORTS

## Security Considerations

- The Plex token should be kept secure and not exposed in logs or error messages
- Consider using environment variables or a secure configuration management system
- The service should be run in a secure environment with appropriate network access controls 