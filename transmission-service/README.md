# Transmission Service

A gRPC service that provides an interface to manage torrents through Transmission.

## Features

- Add torrents using magnet links
- Add torrents using base64 encoded .torrent files
- Get detailed status information for torrents

## Configuration

The service can be configured using the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `TRANSMISSION_HOST` | Transmission RPC host | localhost |
| `TRANSMISSION_PORT` | Transmission RPC port | 9091 |
| `TRANSMISSION_USER` | Transmission RPC username | "" |
| `TRANSMISSION_PASSWORD` | Transmission RPC password | "" |
| `SERVICE_PORT` | gRPC service port | 50051 |

## Building and Running

1. Install dependencies:
```bash
go mod download
```

2. Build the service:
```bash
go build
```

3. Run the service:
```bash
./transmission-service
```

## Example Usage

The service can be used with any gRPC client. Here's an example using `grpcurl`:

```bash
# Add a torrent using magnet link
grpcurl -plaintext -d '{"magnet_link": "magnet:?xt=urn:btih:...", "filedir": "/downloads"}' localhost:50052 transmission.TransmissionService/AddTorrentByMagnet

# Add a torrent using torrent file in base64 format
grpcurl -plaintext -d '{"base64_file": "base64_encoded_torrent_file", "filedir": "/downloads"}' localhost:50052 transmission.TransmissionService/AddTorrentByFile

# Get torrent status
grpcurl -plaintext -d '{"torrent_id": 1}' localhost:50052 transmission.TransmissionService/GetTorrentStatus
```

## API Documentation

The service implements the following gRPC methods:

- `AddTorrentByMagnet`: Add a torrent using a magnet link
- `AddTorrentByFile`: Add a torrent using a base64 encoded .torrent file
- `GetTorrentStatus`: Get detailed status information for a torrent

For detailed API documentation, refer to the proto file in `proto/transmission/transmission-service.proto`. 