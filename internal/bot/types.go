package bot

import (
	"fmt"
	"strconv"
	"time"

	coordinatorpb "github.com/aquare11e/media-downloader-bot/common/protogen/coordinator"
)

type DownloadStatus struct {
	Name     string
	Status   coordinatorpb.DownloadStatus
	Message  string
	ETA      time.Duration
	Progress float64
}

func (d *DownloadStatus) ToRedisMap() map[string]string {
	return map[string]string{
		"name":     d.Name,
		"status":   d.Status.String(),
		"message":  d.Message,
		"eta":      d.ETA.String(),
		"progress": fmt.Sprintf("%f", d.Progress),
	}
}

func (d *DownloadStatus) FromRedisMap(m map[string]string) error {
	d.Name = m["name"]

	status, ok := coordinatorpb.DownloadStatus_value[m["status"]]
	if !ok {
		return fmt.Errorf("invalid status: %s", m["status"])
	}
	d.Status = coordinatorpb.DownloadStatus(status)

	d.Message = m["message"]

	etaStr, ok := m["eta"]
	if !ok {
		return fmt.Errorf("eta not found in map")
	}
	eta, err := time.ParseDuration(etaStr)
	if err != nil {
		return fmt.Errorf("invalid eta: %s", etaStr)
	}
	d.ETA = eta

	progress, err := strconv.ParseFloat(m["progress"], 64)
	if err != nil {
		return fmt.Errorf("invalid progress: %s", m["progress"])
	}
	d.Progress = progress

	return nil
}
