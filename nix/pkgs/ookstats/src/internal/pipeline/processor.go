package pipeline

import (
	"fmt"
	"time"

	"github.com/charmbracelet/log"
)

type BatchConfig struct {
	ComponentName string
	BatchSize     int
	MaxItems      int
	Verbose       bool
}

type BatchCallbacks[T any] struct {
	CountTotal   func() (int, error)
	FetchBatch   func(limit int) ([]T, error)
	ProcessBatch func([]T) error
	ShouldSkip   func(T) bool
	OnComplete   func()
}

type BatchResult struct {
	Processed int
	Duration  time.Duration
}

func RunBatchProcessor[T any](cfg BatchConfig, callbacks BatchCallbacks[T]) (*BatchResult, error) {
	start := time.Now()
	logger := log.With("component", cfg.ComponentName)

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 50
	}

	result := &BatchResult{}

	totalRemaining, err := callbacks.CountTotal()
	var tracker *BatchTracker
	if err != nil {
		logger.Warn(fmt.Sprintf("unable to count %s items", cfg.ComponentName), "error", err)
		tracker = NewBatchTracker(start, -1, cfg.MaxItems)
	} else {
		tracker = NewBatchTracker(start, totalRemaining, cfg.MaxItems)
		logger.Info(fmt.Sprintf("found %s items to process", cfg.ComponentName),
			"count", totalRemaining,
			"target", tracker.DescribeTarget())
	}

	batchNumber := 0
	for {
		if tracker.ShouldStop(result.Processed) {
			break
		}

		currentBatchSize := tracker.AdjustBatchSize(batchSize, result.Processed)
		if currentBatchSize == 0 {
			break
		}

		candidates, err := callbacks.FetchBatch(currentBatchSize)
		if err != nil {
			return nil, err
		}
		if len(candidates) == 0 {
			break
		}

		batchCandidates := make([]T, 0, len(candidates))
		scheduled := 0
		for _, cand := range candidates {
			if callbacks.ShouldSkip != nil && callbacks.ShouldSkip(cand) {
				continue
			}
			if tracker.ShouldStop(result.Processed + scheduled) {
				break
			}
			batchCandidates = append(batchCandidates, cand)
			scheduled++
		}
		if len(batchCandidates) == 0 {
			continue
		}

		logger.Info("processing batch", "batch", batchNumber+1, "batch_size", len(batchCandidates))
		batchNumber++

		if err := callbacks.ProcessBatch(batchCandidates); err != nil {
			return nil, err
		}

		result.Processed += len(batchCandidates)

		logBatchProgress(tracker, logger, result.Processed, batchNumber)
	}

	if callbacks.OnComplete != nil {
		callbacks.OnComplete()
	}

	result.Duration = time.Since(start)
	return result, nil
}
