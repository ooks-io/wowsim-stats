package pipeline

import (
	"fmt"
	"time"
)

type BatchTracker struct {
	maxPlayers int
	target     int
	start      time.Time
	lastLog    time.Time
}

func NewBatchTracker(start time.Time, totalRemaining, maxPlayers int) *BatchTracker {
	target := computeTarget(totalRemaining, maxPlayers)
	return &BatchTracker{
		maxPlayers: maxPlayers,
		target:     target,
		start:      start,
		lastLog:    start,
	}
}

func computeTarget(totalRemaining, maxPlayers int) int {
	if totalRemaining < 0 {
		if maxPlayers > 0 {
			return maxPlayers
		}
		return -1
	}
	if maxPlayers > 0 && (totalRemaining == 0 || maxPlayers < totalRemaining) {
		return maxPlayers
	}
	return totalRemaining
}

func (bt *BatchTracker) Target() int { return bt.target }

func (bt *BatchTracker) DescribeTarget() string {
	if bt.target <= 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%d", bt.target)
}

func (bt *BatchTracker) AdjustBatchSize(batchSize, processed int) int {
	if bt.maxPlayers <= 0 {
		return batchSize
	}
	remaining := bt.maxPlayers - processed
	if remaining <= 0 {
		return 0
	}
	if remaining < batchSize {
		return remaining
	}
	return batchSize
}

func (bt *BatchTracker) ShouldStop(processed int) bool {
	return bt.maxPlayers > 0 && processed >= bt.maxPlayers
}

type BatchProgressStats struct {
	Batch     int
	Processed int
	Target    int
	Remaining int
	Percent   float64
	RPS       float64
	Elapsed   time.Duration
}

func (bt *BatchTracker) ProgressStats(processed, batch int) BatchProgressStats {
	remaining := 0
	if bt.target > 0 {
		remaining = bt.target - processed
		if remaining < 0 {
			remaining = 0
		}
	}
	elapsed := time.Since(bt.start)
	rps := 0.0
	if elapsed > 0 {
		rps = float64(processed) / elapsed.Seconds()
	}
	percent := 0.0
	if bt.target > 0 {
		percent = (float64(processed) / float64(bt.target)) * 100
	}
	return BatchProgressStats{
		Batch:     batch,
		Processed: processed,
		Target:    bt.target,
		Remaining: remaining,
		Percent:   percent,
		RPS:       rps,
		Elapsed:   elapsed,
	}
}

func (bt *BatchTracker) ShouldLogProgress(processed int) bool {
	if bt.target <= 0 {
		return false
	}
	if processed == 0 {
		return false
	}
	if processed%200 == 0 || time.Since(bt.lastLog) >= 30*time.Second {
		bt.lastLog = time.Now()
		return true
	}
	return false
}
