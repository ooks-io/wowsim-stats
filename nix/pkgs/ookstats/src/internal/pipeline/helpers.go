package pipeline

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"ookstats/internal/blizzard"
)

func nowMillis() int64 {
	return time.Now().UnixMilli()
}

func extractCharacterID(resp *blizzard.CharacterStatusResponse) *int {
	if resp == nil || resp.Character.ID <= 0 {
		return nil
	}
	id := resp.Character.ID
	return &id
}

func isNotFoundError(err error) bool {
	var apiErr *blizzard.APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound
}

func logBatchProgress(tracker *BatchTracker, logger *log.Logger, processed, batchNumber int) {
	if !tracker.ShouldLogProgress(processed) {
		return
	}
	stats := tracker.ProgressStats(processed, batchNumber)
	logger.Info("progress",
		"batch", stats.Batch,
		"processed", stats.Processed,
		"target", stats.Target,
		"remaining_estimate", stats.Remaining,
		"percent_complete", fmt.Sprintf("%.1f%%", stats.Percent),
		"req_per_sec", fmt.Sprintf("%.1f", stats.RPS),
		"elapsed", stats.Elapsed.Truncate(time.Second))
}
