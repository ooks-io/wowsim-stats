package blizzard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// PlayerInfo represents basic player information for profile fetching
type PlayerInfo struct {
	ID        int
	Name      string
	RealmSlug string
	Region    string
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

// FetchCharacterStatus fetches the status response for a character (valid/moved/deleted).
func (c *Client) FetchCharacterStatus(playerName, realmSlug, region string) (*CharacterStatusResponse, error) {
	namespace := fmt.Sprintf("profile-classic-%s", region)
	url := fmt.Sprintf(
		"https://%s.api.blizzard.com/profile/wow/character/%s/%s/status?namespace=%s&locale=en_US",
		region, realmSlug, strings.ToLower(playerName), namespace,
	)

	return fetchPlayerProfileAPI[CharacterStatusResponse](c, url)
}

// FetchCharacterAchievements fetches the achievements summary for a character.
func (c *Client) FetchCharacterAchievements(playerName, realmSlug, region string) (*CharacterAchievementsResponse, error) {
	namespace := fmt.Sprintf("profile-classic-%s", region)
	url := fmt.Sprintf(
		"https://%s.api.blizzard.com/profile/wow/character/%s/%s/achievements?namespace=%s&locale=en_US",
		region, realmSlug, strings.ToLower(playerName), namespace,
	)

	return fetchPlayerProfileAPI[CharacterAchievementsResponse](c, url)
}

// fetchPlayerProfileAPI is a generic function for fetching player profile data
func fetchPlayerProfileAPI[T any](c *Client, url string) (*T, error) {
	const maxRetries = 3
	const baseDelay = 1 * time.Second

	var lastErr error
	attempt := 0
	for {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * baseDelay
			time.Sleep(delay)
		}

		result, err := fetchPlayerProfileAPIOnce[T](c, url)
		if err == nil {
			return result, nil
		}

		lastErr = err

		var apiErr *APIError
		if errors.As(err, &apiErr) {
			switch apiErr.Status {
			case http.StatusTooManyRequests:
				time.Sleep(apiErr.retryDelay())
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

// fetchPlayerProfileAPIOnce performs a single player profile API fetch attempt
func fetchPlayerProfileAPIOnce[T any](c *Client, url string) (*T, error) {
	c.waitForRateSlot()

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
		return nil, newAPIError(resp.StatusCode, bodyBytes, resp.Header.Get("Retry-After"))
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// FetchPlayerProfilesConcurrent fetches player profiles concurrently with rate limiting
func (c *Client) FetchPlayerProfilesConcurrent(ctx context.Context, players []PlayerInfo) <-chan PlayerProfileResult {
	results := make(chan PlayerProfileResult, len(players))
	var wg sync.WaitGroup

	for _, player := range players {
		wg.Add(1)
		go func(p PlayerInfo) {
			defer wg.Done()

			// Acquire concurrency slot
			select {
			case c.concurrency <- struct{}{}:
				defer func() { <-c.concurrency }() // Release slot
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
