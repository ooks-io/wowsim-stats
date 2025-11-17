package blizzard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Client struct {
    HTTPClient  *http.Client
    Token       string
    rateLimiter chan struct{} // semaphore for rate limiting
    // Verbose controls extra per-request logging
    Verbose bool
    // metrics
    reqCount       int64
    notFoundCount  int64
    totalLatencyMs int64
}

// FetchResult represents the result of a fetch operation
type FetchResult struct {
	RealmInfo   RealmInfo
	Dungeon     DungeonInfo
	Leaderboard *LeaderboardResponse
	Error       error
}

// NewClient creates a new Blizzard API client
func NewClient() (*Client, error) {
	token := getEnvOrFail("BLIZZARD_API_TOKEN")

	// configure hhtp client with connection pooling
	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		MaxIdleConnsPerHost: 10,
	}

	// create rate limiter with 20 concurrent requests max
	rateLimiter := make(chan struct{}, 20)

    client := &Client{
        HTTPClient: &http.Client{
            Timeout:   15 * time.Second,
            Transport: transport,
        },
        Token:       token,
        rateLimiter: rateLimiter,
    }

    return client, nil
}

// SetConcurrency adjusts the maximum concurrent API requests.
func (c *Client) SetConcurrency(n int) {
    if n <= 0 {
        n = 1
    }
    c.rateLimiter = make(chan struct{}, n)
}

// SetTimeout updates the HTTP client timeout.
func (c *Client) SetTimeout(d time.Duration) {
    if d <= 0 {
        return
    }
    if c.HTTPClient != nil {
        c.HTTPClient.Timeout = d
    }
}

// FetchLeaderboardData fetches leaderboard data for a specific realm and dungeon with retries
func (c *Client) FetchLeaderboardData(realmInfo RealmInfo, dungeon DungeonInfo, periodID string) (*LeaderboardResponse, error) {
	const maxRetries = 3
	const baseDelay = 1 * time.Second

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// exponential backoff with jitter
			delay := time.Duration(1<<uint(attempt-1)) * baseDelay
			time.Sleep(delay)
			fmt.Printf("    [RETRY %d/%d] Retrying after %v delay...\n", attempt+1, maxRetries, delay)
		}

        result, err := c.fetchLeaderboardDataOnce(realmInfo, dungeon, periodID)
        if err == nil {
            return result, nil
        }

		lastErr = err

        // dont retry on certain errors (4xx client errors)
        if strings.Contains(err.Error(), "status 4") {
            if c.Verbose {
                fmt.Printf("    [ERROR] Client error, not retrying: %v\n", err)
            }
            break
        }
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// fetchLeaderboardDataOnce performs a single fetch attempt
func (c *Client) fetchLeaderboardDataOnce(realmInfo RealmInfo, dungeon DungeonInfo, periodID string) (*LeaderboardResponse, error) {
    // rate limiting - 50ms delay between requests
    time.Sleep(50 * time.Millisecond)

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
        return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
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

// Stats returns simple client-side metrics for diagnostics
func (c *Client) Stats() (requests int64, notFound int64, avgLatencyMs float64) {
    req := atomic.LoadInt64(&c.reqCount)
    nf := atomic.LoadInt64(&c.notFoundCount)
    tot := atomic.LoadInt64(&c.totalLatencyMs)
    var avg float64
    if req > 0 {
        avg = float64(tot) / float64(req)
    }
    return req, nf, avg
}

// FetchLeaderboardsConcurrent fetches multiple leaderboards concurrently
func (c *Client) FetchLeaderboardsConcurrent(ctx context.Context, realmInfo RealmInfo, dungeons []DungeonInfo, periodID string) <-chan FetchResult {
	results := make(chan FetchResult, len(dungeons))
	var wg sync.WaitGroup

	for _, dungeon := range dungeons {
		wg.Add(1)
		go func(d DungeonInfo) {
			defer wg.Done()

			// Acquire rate limiter token
			select {
			case c.rateLimiter <- struct{}{}:
				defer func() { <-c.rateLimiter }() // Release token
			case <-ctx.Done():
				results <- FetchResult{
					RealmInfo: realmInfo,
					Dungeon:   d,
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

// FetchCharacterSummary fetches character summary data
func (c *Client) FetchCharacterSummary(playerName, realmSlug, region string) (*CharacterSummaryResponse, error) {
	namespace := fmt.Sprintf("profile-classic-%s", region)
	url := fmt.Sprintf(
		"https://%s.api.blizzard.com/profile/wow/character/%s/%s?namespace=%s&locale=en_US",
		region, realmSlug, strings.ToLower(playerName), namespace,
	)

	return fetchPlayerProfileAPI[CharacterSummaryResponse](c, url)
}

// FetchCharacterEquipment fetches character equipment data
func (c *Client) FetchCharacterEquipment(playerName, realmSlug, region string) (*CharacterEquipmentResponse, error) {
	namespace := fmt.Sprintf("profile-classic-%s", region)
	url := fmt.Sprintf(
		"https://%s.api.blizzard.com/profile/wow/character/%s/%s/equipment?namespace=%s&locale=en_US",
		region, realmSlug, strings.ToLower(playerName), namespace,
	)

	return fetchPlayerProfileAPI[CharacterEquipmentResponse](c, url)
}

// FetchCharacterMedia fetches character media data (avatars)
func (c *Client) FetchCharacterMedia(playerName, realmSlug, region string) (*CharacterMediaResponse, error) {
	namespace := fmt.Sprintf("profile-classic-%s", region)
	url := fmt.Sprintf(
		"https://%s.api.blizzard.com/profile/wow/character/%s/%s/character-media?namespace=%s&locale=en_US",
		region, realmSlug, strings.ToLower(playerName), namespace,
	)

	return fetchPlayerProfileAPI[CharacterMediaResponse](c, url)
}

// fetchPlayerProfileAPI is a generic function for fetching player profile data
func fetchPlayerProfileAPI[T any](c *Client, url string) (*T, error) {
	const maxRetries = 3
	const baseDelay = 1 * time.Second

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * baseDelay
			time.Sleep(delay)
		}

		result, err := fetchPlayerProfileAPIOnce[T](c, url)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// don't retry on certain errors (4xx client errors)
		if strings.Contains(err.Error(), "status 4") {
			break
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// fetchPlayerProfileAPIOnce performs a single player profile API fetch attempt
func fetchPlayerProfileAPIOnce[T any](c *Client, url string) (*T, error) {
	// Rate limiting - 50ms delay between requests
	time.Sleep(50 * time.Millisecond)

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
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// PlayerProfileResult represents the result of a player profile fetch operation
type PlayerProfileResult struct {
	PlayerID   int
	PlayerName string
	RealmSlug  string
	Region     string
	Summary    *CharacterSummaryResponse
	Equipment  *CharacterEquipmentResponse
	Media      *CharacterMediaResponse
	Error      error
}

// FetchPlayerProfilesConcurrent fetches player profiles concurrently with rate limiting
func (c *Client) FetchPlayerProfilesConcurrent(ctx context.Context, players []PlayerInfo) <-chan PlayerProfileResult {
	results := make(chan PlayerProfileResult, len(players))
	var wg sync.WaitGroup

	for _, player := range players {
		wg.Add(1)
		go func(p PlayerInfo) {
			defer wg.Done()

			// Acquire rate limiter token
			select {
			case c.rateLimiter <- struct{}{}:
				defer func() { <-c.rateLimiter }() // Release token
			case <-ctx.Done():
				results <- PlayerProfileResult{
					PlayerID:   p.ID,
					PlayerName: p.Name,
					RealmSlug:  p.RealmSlug,
					Region:     p.Region,
					Error:      ctx.Err(),
				}
				return
			}

			// minimal delay to respect API rate limits
			time.Sleep(20 * time.Millisecond)

			// fetch all profile data concurrently
			var summary *CharacterSummaryResponse
			var equipment *CharacterEquipmentResponse
			var media *CharacterMediaResponse
			var profileErr error

			// use a sub-context for the individual fetches
			subCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			// launch concurrent fetches for this player
			summaryDone := make(chan struct{})
			equipmentDone := make(chan struct{})
			mediaDone := make(chan struct{})

			go func() {
				defer close(summaryDone)
				var err error
				summary, err = c.FetchCharacterSummary(p.Name, p.RealmSlug, p.Region)
				if err != nil && profileErr == nil {
					profileErr = fmt.Errorf("summary fetch failed: %w", err)
				}
			}()

			go func() {
				defer close(equipmentDone)
				var err error
				equipment, err = c.FetchCharacterEquipment(p.Name, p.RealmSlug, p.Region)
				if err != nil && profileErr == nil {
					profileErr = fmt.Errorf("equipment fetch failed: %w", err)
				}
			}()

			go func() {
				defer close(mediaDone)
				var err error
				media, err = c.FetchCharacterMedia(p.Name, p.RealmSlug, p.Region)
				if err != nil && profileErr == nil {
					profileErr = fmt.Errorf("media fetch failed: %w", err)
				}
			}()

			// wait for all fetches to complete or timeout
			select {
			case <-summaryDone:
			case <-subCtx.Done():
				profileErr = fmt.Errorf("summary fetch timeout")
			}

			select {
			case <-equipmentDone:
			case <-subCtx.Done():
				if profileErr == nil {
					profileErr = fmt.Errorf("equipment fetch timeout")
				}
			}

			select {
			case <-mediaDone:
			case <-subCtx.Done():
				if profileErr == nil {
					profileErr = fmt.Errorf("media fetch timeout")
				}
			}

			results <- PlayerProfileResult{
				PlayerID:   p.ID,
				PlayerName: p.Name,
				RealmSlug:  p.RealmSlug,
				Region:     p.Region,
				Summary:    summary,
				Equipment:  equipment,
				Media:      media,
				Error:      profileErr,
			}
		}(player)
	}

	// close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// PlayerInfo represents basic player information for profile fetching
type PlayerInfo struct {
	ID        int
	Name      string
	RealmSlug string
	Region    string
}

// FetchSeasonIndex fetches the list of available seasons for a region
func (c *Client) FetchSeasonIndex(region string) (*SeasonIndexResponse, error) {
	namespace := fmt.Sprintf("dynamic-classic-%s", region)
	url := fmt.Sprintf(
		"https://%s.api.blizzard.com/data/wow/mythic-keystone/season/index?namespace=%s&locale=en_US",
		region, namespace,
	)

	const maxRetries = 3
	const baseDelay = 1 * time.Second

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * baseDelay
			time.Sleep(delay)
		}

		result, err := c.fetchSeasonIndexOnce(url)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if strings.Contains(err.Error(), "status 4") {
			break
		}
	}

	return nil, fmt.Errorf("failed to fetch season index after %d attempts: %w", maxRetries, lastErr)
}

func (c *Client) fetchSeasonIndexOnce(url string) (*SeasonIndexResponse, error) {
	time.Sleep(50 * time.Millisecond)

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
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result SeasonIndexResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// FetchSeasonDetail fetches details for a specific season
func (c *Client) FetchSeasonDetail(region string, seasonID int) (*SeasonDetailResponse, error) {
	namespace := fmt.Sprintf("dynamic-classic-%s", region)
	url := fmt.Sprintf(
		"https://%s.api.blizzard.com/data/wow/mythic-keystone/season/%d?namespace=%s&locale=en_US",
		region, seasonID, namespace,
	)

	const maxRetries = 3
	const baseDelay = 1 * time.Second

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * baseDelay
			time.Sleep(delay)
		}

		result, err := c.fetchSeasonDetailOnce(url)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if strings.Contains(err.Error(), "status 4") {
			break
		}
	}

	return nil, fmt.Errorf("failed to fetch season %d after %d attempts: %w", seasonID, maxRetries, lastErr)
}

func (c *Client) fetchSeasonDetailOnce(url string) (*SeasonDetailResponse, error) {
	time.Sleep(50 * time.Millisecond)

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
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result SeasonDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func getEnvOrFail(key string) string {
	value := os.Getenv(key)
	if value == "" {
		fmt.Fprintf(os.Stderr, "Error: %s environment variable is required\n", key)
		os.Exit(1)
	}
	return value
}
