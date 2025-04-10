# Media Downloader Bot

A distributed system for managing media downloads through Telegram, with automatic Plex integration.

## Services

The system consists of four microservices:

1. **Bot Service** (`/bot`)
   - Telegram bot interface
   - Handles user commands and authentication
   - Communicates with coordinator service

2. **Coordinator Service** (`/coordinator-service`)
   - Central service that coordinates all operations
   - Manages download requests
   - Handles communication between services
   - Uses Redis for state management

3. **Plex Service** (`/plex-service`)
   - Updates Plex library

4. **Transmission Service** (`/transmission-service`)
   - Handles download operations
   - Provides download status updates

## Prerequisites

- Go 1.23 or later
- Docker and Docker Compose
- protoc (Protocol Buffers compiler)
- Redis
- Plex Media Server
- Transmission torrent client

## Development Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/aquare11e/media-downloader-bot.git
   cd media-downloader-bot
   ```

2. Generate protobuf files:
   ```bash
   make protoc
   ```

3. Set up environment variables:
   - Copy `.env.example` to `.env` in each service directory
   - Fill in the required environment variables

4. Build and run services:
   ```bash
   docker-compose up -d
   ```

## Environment Variables

### Bot Service
- `TELEGRAM_BOT_TOKEN`: Telegram bot token
- `ALLOWED_USERS`: Comma-separated list of allowed Telegram user IDs
- `COORDINATOR_SERVICE_URL`: URL of the coordinator service

### Coordinator Service
- `SERVICE_PORT`: The port number on which the gRPC server will listen.
- `TRANSMISSION_SERVICE_URL`: The URL of the Transmission service.
- `PLEX_SERVICE_URL`: URL of the Plex service
- `REDIS_URL`: Redis connection URL
- `REDIS_PASSWORD`: Redis password
- `*_DIR_PATH`: Paths for different media types

### Plex Service
- `SERVICE_PORT`: gRPC service port
- `PLEX_HOST`: Plex server host
- `PLEX_PORT`: Plex server port
- `PLEX_TOKEN`: Plex authentication token
- `PLEX_CATEGORY_*`: Category IDs for different media types

### Transmission Service
- `SERVICE_PORT`: gRPC service port
- `TRANSMISSION_HOST`: Transmission host
- `TRANSMISSION_PORT`: Transmission port
- `TRANSMISSION_USER`: Transmission username
- `TRANSMISSION_PASSWORD`: Transmission password

## Docker Images

Docker images are automatically built and pushed to GitHub Container Registry (GHCR) on each push to main branch.

Images are available at:
- `ghcr.io/aquare11e/media-downloader-bot/bot:latest`
- `ghcr.io/aquare11e/media-downloader-bot/coordinator:latest`
- `ghcr.io/aquare11e/media-downloader-bot/plex:latest`
- `ghcr.io/aquare11e/media-downloader-bot/transmission:latest`

## Usage

1. Start a conversation with your Telegram bot
2. Send `/start` to begin
3. Available commands:
   - `/download` - Start a download
   - `/status` - Check the current status of ongoing downloads
   - `/help` - Get a list of available commands and their descriptions

## License

This project is licensed under the MIT License - see the LICENSE file for details. 