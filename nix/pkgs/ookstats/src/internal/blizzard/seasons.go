package blizzard

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

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
	attempt := 0
	for {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * baseDelay
			time.Sleep(delay)
		}

		result, err := c.fetchSeasonIndexOnce(url)
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

	return nil, fmt.Errorf("failed to fetch season index after %d attempts: %w", maxRetries, lastErr)
}

func (c *Client) fetchSeasonIndexOnce(url string) (*SeasonIndexResponse, error) {
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
	attempt := 0
	for {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * baseDelay
			time.Sleep(delay)
		}

		result, err := c.fetchSeasonDetailOnce(url)
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

	return nil, fmt.Errorf("failed to fetch season %d after %d attempts: %w", seasonID, maxRetries, lastErr)
}

func (c *Client) fetchSeasonDetailOnce(url string) (*SeasonDetailResponse, error) {
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

	var result SeasonDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
