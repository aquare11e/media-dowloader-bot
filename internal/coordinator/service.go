package coordinator

import (
	"context"
	"fmt"
	"log"
	"time"

	common "github.com/aquare11e/media-dowloader-bot/common/protogen/common"
	coordinatorpb "github.com/aquare11e/media-dowloader-bot/common/protogen/coordinator"
	"github.com/aquare11e/media-dowloader-bot/common/protogen/plex"
	"github.com/aquare11e/media-dowloader-bot/common/protogen/transmission"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	coordinatorpb.UnimplementedCoordinatorServiceServer
	transmissionClient   transmission.TransmissionServiceClient
	plexClient           plex.PlexServiceClient
	redisClient          *redis.Client
	pbTypeToDownloadPath map[common.RequestType]string
}

func NewService(transmissionConn, plexConn *grpc.ClientConn, redisClient *redis.Client, pbTypeToDownloadPath map[common.RequestType]string) *Service {
	return &Service{
		transmissionClient:   transmission.NewTransmissionServiceClient(transmissionConn),
		plexClient:           plex.NewPlexServiceClient(plexConn),
		redisClient:          redisClient,
		pbTypeToDownloadPath: pbTypeToDownloadPath,
	}
}

func (s *Service) AddTorrentByMagnet(req *coordinatorpb.AddTorrentByMagnetRequest, stream coordinatorpb.CoordinatorService_AddTorrentByMagnetServer) error {
	ctx := stream.Context()
	requestID := uuid.New().String()

	log.Printf("Adding torrent by magnet (requestID: %s, category: %s)", requestID, req.Category)

	// Add torrent to Transmission
	transmissionReq := &transmission.AddTorrentByMagnetRequest{
		MagnetLink: req.MagnetLink,
		Filedir:    s.pbTypeToDownloadPath[req.Category],
	}

	resp, err := s.transmissionClient.AddTorrentByMagnet(ctx, transmissionReq)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to add torrent: %v", err)
	}

	// Save to Redis
	torrentRecord := &TorrentRecord{
		RequestID:  requestID,
		Category:   req.Category,
		LastUpdate: time.Now(),
	}
	err = s.redisClient.HSet(ctx, fmt.Sprintf(KeyTorrentFormat, resp.TorrentId), torrentRecord.ToRedisMap()).Err()
	if err != nil {
		return status.Errorf(codes.Internal, "failed to save to Redis: %v", err)
	}

	// Send initial response
	if err := stream.Send(&coordinatorpb.DownloadResponse{
		RequestId: requestID,
		Status:    coordinatorpb.DownloadStatus_DOWNLOAD_STATUS_IN_PROGRESS,
		Message:   "Download started",
	}); err != nil {
		return err
	}

	// Start streaming status updates
	return s.streamDownloadStatus(ctx, requestID, resp.TorrentId, req.Category, stream)
}

func (s *Service) AddTorrentByFile(req *coordinatorpb.AddTorrentByFileRequest, stream coordinatorpb.CoordinatorService_AddTorrentByFileServer) error {
	ctx := stream.Context()
	requestID := uuid.New().String()

	log.Printf("Adding torrent by file (requestID: %s, category: %s)", requestID, req.Category)

	// Add torrent to Transmission
	transmissionReq := &transmission.AddTorrentByFileRequest{
		Base64File: req.Base64File,
		Filedir:    s.pbTypeToDownloadPath[req.Category],
	}

	resp, err := s.transmissionClient.AddTorrentByFile(ctx, transmissionReq)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to add torrent: %v", err)
	}

	// Save to Redis
	torrentRecord := &TorrentRecord{
		RequestID:  requestID,
		Category:   req.Category,
		LastUpdate: time.Now(),
	}
	err = s.redisClient.HSet(ctx, fmt.Sprintf(KeyTorrentFormat, resp.TorrentId), torrentRecord.ToRedisMap()).Err()
	if err != nil {
		return status.Errorf(codes.Internal, "failed to save to Redis: %v", err)
	}

	// Send initial response
	if err := stream.Send(&coordinatorpb.DownloadResponse{
		RequestId: requestID,
		Status:    coordinatorpb.DownloadStatus_DOWNLOAD_STATUS_IN_PROGRESS,
		Message:   "Download started",
	}); err != nil {
		return err
	}

	// Start streaming status updates
	return s.streamDownloadStatus(ctx, requestID, resp.TorrentId, req.Category, stream)
}

func (s *Service) streamDownloadStatus(ctx context.Context, requestID string, torrentID int64, category common.RequestType, stream any) error {
	for {
		// Get torrent status
		statusReq := &transmission.GetTorrentStatusRequest{
			TorrentId: torrentID,
		}
		statusResp, err := s.transmissionClient.GetTorrentStatus(ctx, statusReq)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to get torrent status: %v", err)
		}

		if statusResp.Status == transmission.TorrentStatus_STATUS_ERROR {
			return status.Errorf(codes.Internal, "torrent status is error, check transmission's download: %s", statusResp.Name)
		}

		log.Printf("Torrent status (id: %d): %s", torrentID, statusResp.Status)

		// Update response
		response := &coordinatorpb.DownloadResponse{
			RequestId: requestID,
			Progress:  statusResp.Progress,
			Eta:       int32(statusResp.Eta),
		}

		// Check if download is complete
		if statusResp.Status == transmission.TorrentStatus_STATUS_DONE {
			// Refresh Plex library
			plexReq := &plex.UpdateCategoryRequest{
				RequestId: requestID,
				Type:      category,
			}
			plexResp, err := s.plexClient.UpdateCategory(ctx, plexReq)
			if err != nil {
				response.Status = coordinatorpb.DownloadStatus_DOWNLOAD_STATUS_ERROR
				response.Message = fmt.Sprintf("failed to refresh Plex library: %v", err)
			} else if plexResp.Result == plex.ResponseResult_RESPONSE_RESULT_SUCCESS {
				response.Status = coordinatorpb.DownloadStatus_DOWNLOAD_STATUS_SUCCESS
				response.Message = "Download completed and library refreshed"
			} else {
				response.Status = coordinatorpb.DownloadStatus_DOWNLOAD_STATUS_ERROR
				response.Message = plexResp.Message
			}

			// Clean up Redis
			s.redisClient.Del(ctx, fmt.Sprintf(KeyTorrentFormat, torrentID))

			// Send final response and return
			if err := sendResponse(stream, response); err != nil {
				return err
			}
			return nil
		}

		// Send progress update
		if err := sendResponse(stream, response); err != nil {
			return err
		}

		// Update last update time
		record := &TorrentRecord{
			RequestID:  requestID,
			Category:   category,
			LastUpdate: time.Now(),
		}
		err = s.redisClient.HSet(ctx, fmt.Sprintf(KeyTorrentFormat, torrentID), record.ToRedisMap()).Err()
		if err != nil {
			return status.Errorf(codes.Internal, "failed to update last update time: %v", err)
		}

		// Wait based on ETA
		waitTime := time.Duration(statusResp.Eta)*time.Second + EtaErrorSeconds*time.Second
		if waitTime > 0 && waitTime < time.Minute {
			time.Sleep(waitTime)
		} else {
			time.Sleep(time.Minute)
		}
	}
}

func sendResponse(stream any, response *coordinatorpb.DownloadResponse) error {
	if s, ok := stream.(grpc.ServerStreamingServer[coordinatorpb.DownloadResponse]); ok {
		return s.Send(response)
	}
	return status.Errorf(codes.Internal, "invalid stream type")
}
