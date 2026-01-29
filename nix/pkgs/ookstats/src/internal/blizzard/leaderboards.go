package blizzard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// FetchResult represents the result of a fetch operation
type FetchResult struct {
	RealmInfo   RealmInfo
	Dungeon     DungeonInfo
	PeriodID    string
	Leaderboard *LeaderboardResponse
	Error       error
}

// FetchLeaderboardData fetches leaderboard data for a specific realm and dungeon with retries
func (c *Client) FetchLeaderboardData(realmInfo RealmInfo, dungeon DungeonInfo, periodID string) (*LeaderboardResponse, error) {
	const maxRetries = 3
	const baseDelay = 1 * time.Second

	var lastErr error
	attempt := 0
	for {
		if attempt > 0 {
			// exponential backoff between full retries (non-429)
			delay := time.Duration(1<<uint(attempt-1)) * baseDelay
			time.Sleep(delay)
			if c.Verbose {
				fmt.Printf("    [RETRY %d/%d] Retrying after %v delay...\n", attempt, maxRetries, delay)
			}
		}

		result, err := c.fetchLeaderboardDataOnce(realmInfo, dungeon, periodID)
		if err == nil {
			return result, nil
		}

		lastErr = err

		var apiErr *APIError
		if errors.As(err, &apiErr) {
			switch apiErr.Status {
			case http.StatusTooManyRequests:
				delay := apiErr.retryDelay()
				if c.Verbose {
					fmt.Printf("    [WARN] 429 received, backing off for %v\n", delay)
				}
				time.Sleep(delay)
				continue
			case http.StatusNotFound:
				return nil, err
			}
		}

		attempt++
		if attempt >= maxRetries {
			break
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// fetchLeaderboardDataOnce performs a single fetch attempt
func (c *Client) fetchLeaderboardDataOnce(realmInfo RealmInfo, dungeon DungeonInfo, periodID string) (*LeaderboardResponse, error) {
	c.waitForRateSlot()

	start := time.Now()

	region := realmInfo.Region
	realmID := realmInfo.ID
	dungeonID := dungeon.ID

	namespace := fmt.Sprintf("dynamic-classic-%s", region)
	url := fmt.Sprintf(
		"https://%s.api.blizzard.com/data/wow/connected-realm/%d/mythic-leaderboard/%d/period/%s?namespace=%s",
		region, realmID, dungeonID, periodID, namespace,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	req.Header.Set("User-Agent", "WoWStatsDB/1.0")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		// metrics
		atomic.AddInt64(&c.reqCount, 1)
		if resp.StatusCode == http.StatusNotFound {
			atomic.AddInt64(&c.notFoundCount, 1)
		}
		atomic.AddInt64(&c.totalLatencyMs, time.Since(start).Milliseconds())
		if c.Verbose {
			fmt.Printf("HTTP %d %-3s %-20s %-26s in %dms\n", resp.StatusCode, realmInfo.Region, realmInfo.Name, dungeon.Name, time.Since(start).Milliseconds())
		}
		return nil, newAPIError(resp.StatusCode, bodyBytes, resp.Header.Get("Retry-After"))
	}

	var leaderboard LeaderboardResponse
	if err := json.NewDecoder(resp.Body).Decode(&leaderboard); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// successfully decoded
	atomic.AddInt64(&c.reqCount, 1)
	atomic.AddInt64(&c.totalLatencyMs, time.Since(start).Milliseconds())
	if c.Verbose {
		fmt.Printf("HTTP 200 %-3s %-20s %-26s in %dms\n", realmInfo.Region, realmInfo.Name, dungeon.Name, time.Since(start).Milliseconds())
	}
	return &leaderboard, nil
}

// FetchLeaderboardsConcurrent fetches multiple leaderboards concurrently
func (c *Client) FetchLeaderboardsConcurrent(ctx context.Context, realmInfo RealmInfo, dungeons []DungeonInfo, periodID string) <-chan FetchResult {
	results := make(chan FetchResult, len(dungeons))
	var wg sync.WaitGroup

	for _, dungeon := range dungeons {
		wg.Add(1)
		go func(d DungeonInfo) {
			defer wg.Done()

			// Acquire concurrency slot
			select {
			case c.concurrency <- struct{}{}:
				defer func() { <-c.concurrency }() // Release slot
			case <-ctx.Done():
				results <- FetchResult{
					RealmInfo: realmInfo,
					Dungeon:   d,
					PeriodID:  periodID,
					Error:     ctx.Err(),
				}
				return
			}

			// minimal delay to respect API rate limits
			time.Sleep(20 * time.Millisecond)

			leaderboard, err := c.FetchLeaderboardData(realmInfo, d, periodID)
			results <- FetchResult{
				RealmInfo:   realmInfo,
				Dungeon:     d,
				PeriodID:    periodID,
				Leaderboard: leaderboard,
				Error:       err,
			}
		}(dungeon)
	}

	// close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// FetchAllRealmsConcurrent fetches leaderboards for multiple realms concurrently
func (c *Client) FetchAllRealmsConcurrent(ctx context.Context, realms map[string]RealmInfo, dungeons []DungeonInfo, periodID string) <-chan FetchResult {
	results := make(chan FetchResult, len(realms)*len(dungeons))
	var wg sync.WaitGroup

	for realmSlug, realmInfo := range realms {
		wg.Add(1)
		go func(slug string, info RealmInfo) {
			defer wg.Done()

			fmt.Printf("Processing Realm: %s (%s)\n", info.Name, info.Region)

			// fetch all dungeons for this realm concurrently
			realmResults := c.FetchLeaderboardsConcurrent(ctx, info, dungeons, periodID)

			// forward results from this realm
			for result := range realmResults {
				select {
				case results <- result:
				case <-ctx.Done():
					return
				}
			}
		}(realmSlug, realmInfo)
	}

	// close channel when all realms complete
	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}
