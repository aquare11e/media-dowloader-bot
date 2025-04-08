package transmission

import (
	"context"
	"fmt"

	transmissionpb "github.com/aquare11e/media-dowloader-bot/common/protogen/transmission"
	transmissionrpc "github.com/hekmon/transmissionrpc/v3"
)

type Server struct {
	transmissionpb.UnimplementedTransmissionServiceServer
	client *transmissionrpc.Client
}

func NewServer(client *transmissionrpc.Client) *Server {
	return &Server{
		client: client,
	}
}

func (s *Server) AddTorrentByMagnet(ctx context.Context, req *transmissionpb.AddTorrentByMagnetRequest) (*transmissionpb.AddTorrentResponse, error) {
	paused := false
	payload := &transmissionrpc.TorrentAddPayload{
		Filename:    &req.MagnetLink,
		DownloadDir: &req.Filedir,
		Paused:      &paused,
	}

	torrent, err := s.client.TorrentAdd(ctx, *payload)
	if err != nil {
		return nil, fmt.Errorf("failed to add torrent: %v", err)
	}

	return &transmissionpb.AddTorrentResponse{
		TorrentId: *torrent.ID,
	}, nil
}

func (s *Server) AddTorrentByFile(ctx context.Context, req *transmissionpb.AddTorrentByFileRequest) (*transmissionpb.AddTorrentResponse, error) {
	paused := false
	payload := &transmissionrpc.TorrentAddPayload{
		MetaInfo:    &req.Base64File,
		DownloadDir: &req.Filedir,
		Paused:      &paused,
	}

	torrent, err := s.client.TorrentAdd(ctx, *payload)
	if err != nil {
		return nil, fmt.Errorf("failed to add torrent: %v", err)
	}

	return &transmissionpb.AddTorrentResponse{
		TorrentId: *torrent.ID,
	}, nil
}

func (s *Server) GetTorrentStatus(ctx context.Context, req *transmissionpb.GetTorrentStatusRequest) (*transmissionpb.GetTorrentStatusResponse, error) {
	torrent, err := s.client.TorrentGet(ctx, fields, []int64{req.TorrentId})
	if err != nil {
		return nil, fmt.Errorf("failed to get torrent status: %v", err)
	}

	if len(torrent) == 0 {
		return nil, fmt.Errorf("torrent not found")
	}

	t := torrent[0]
	status := transmissionpb.TorrentStatus_STATUS_UNSPECIFIED

	switch {
	case *t.Status == transmissionrpc.TorrentStatusStopped:
		status = transmissionpb.TorrentStatus_STATUS_STOPPED
	case *t.Status == transmissionrpc.TorrentStatusCheckWait:
		status = transmissionpb.TorrentStatus_STATUS_IN_PROGRESS
	case *t.Status == transmissionrpc.TorrentStatusCheck:
		status = transmissionpb.TorrentStatus_STATUS_IN_PROGRESS
	case *t.Status == transmissionrpc.TorrentStatusDownloadWait:
		status = transmissionpb.TorrentStatus_STATUS_IN_PROGRESS
	case *t.Status == transmissionrpc.TorrentStatusDownload:
		status = transmissionpb.TorrentStatus_STATUS_IN_PROGRESS
	case *t.Status == transmissionrpc.TorrentStatusSeedWait:
		status = transmissionpb.TorrentStatus_STATUS_DONE
	case *t.Status == transmissionrpc.TorrentStatusSeed:
		status = transmissionpb.TorrentStatus_STATUS_DONE
	case *t.Status == transmissionrpc.TorrentStatusIsolated:
		status = transmissionpb.TorrentStatus_STATUS_ERROR
	}

	return &transmissionpb.GetTorrentStatusResponse{
		TorrentId:       *t.ID,
		Name:            *t.Name,
		Progress:        *t.PercentDone * 100,
		SizeBytes:       int64(*t.TotalSize),
		DownloadedBytes: *t.HaveValid,
		UploadedBytes:   *t.HaveUnchecked,
		Status:          status,
		DownloadRate:    int32(*t.RateDownload),
		Eta:             int32(*t.ETA),
	}, nil
}

var fields = []string{"id", "status", "name", "percentDone", "totalSize", "haveValid", "haveUnchecked", "rateDownload", "eta"}
