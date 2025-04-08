OUT_DIR=./common/protogen

PROTO_PATH=./proto
COMMON_TYPES_PROTO_FILE=common/common-types.proto
PLEX_PROTO_FILE=plex/plex-service.proto
TRANSMISSION_PROTO_FILE=transmission/transmission-service.proto
COORDINATOR_PROTO_FILE=coordinator/coordinator-service.proto

protoc-common:
	protoc --go_out=$(OUT_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(OUT_DIR) --go-grpc_opt=paths=source_relative \
		--proto_path=$(PROTO_PATH) \
		$(COMMON_TYPES_PROTO_FILE)

protoc-plex:
	protoc --go_out=$(OUT_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(OUT_DIR) --go-grpc_opt=paths=source_relative \
		--proto_path=$(PROTO_PATH) \
		$(PLEX_PROTO_FILE)

protoc-transmission:
	protoc --go_out=$(OUT_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(OUT_DIR) --go-grpc_opt=paths=source_relative \
		--proto_path=$(PROTO_PATH) \
		$(TRANSMISSION_PROTO_FILE)

protoc-coordinator:
	protoc --go_out=$(OUT_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(OUT_DIR) --go-grpc_opt=paths=source_relative \
		--proto_path=$(PROTO_PATH) \
		$(COORDINATOR_PROTO_FILE)

protoc:
	mkdir -p $(OUT_DIR)
	$(MAKE) protoc-common
	$(MAKE) protoc-plex
	$(MAKE) protoc-transmission
	$(MAKE) protoc-coordinator

clean:
	go clean -modcache
	rm -rf $(OUT_DIR)
