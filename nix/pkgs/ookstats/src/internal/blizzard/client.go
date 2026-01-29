package blizzard

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Client struct {
	HTTPClient  *http.Client
	Token       string
	concurrency chan struct{}
	rateTicker  *time.Ticker
	rateMu      sync.Mutex
	ratePrimed  bool
	// Verbose controls extra per-request logging
	Verbose bool
	// metrics
	reqCount       int64
	notFoundCount  int64
	totalLatencyMs int64
}

const (
	defaultConcurrency          = 20
	DefaultRequestRatePerSecond = 90
	minRatePerSecond            = 1
)

// NewClient creates a new Blizzard API client
func NewClient() (*Client, error) {
	token := getEnvOrFail("BLIZZARD_API_TOKEN")

	// configure hhtp client with connection pooling
	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		MaxIdleConnsPerHost: 10,
	}

	// create concurrency limiter with default slots
	concurrency := make(chan struct{}, defaultConcurrency)

	client := &Client{
		HTTPClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
		Token:       token,
		concurrency: concurrency,
	}
	client.setRequestRate(DefaultRequestRatePerSecond)

	return client, nil
}

// SetConcurrency adjusts the maximum concurrent API requests.
func (c *Client) SetConcurrency(n int) {
	if n <= 0 {
		n = 1
	}
	c.concurrency = make(chan struct{}, n)
}

// SetRequestRate updates the max requests per second.
func (c *Client) SetRequestRate(rps int) {
	c.setRequestRate(rps)
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

func (c *Client) setRequestRate(rps int) {
	c.rateMu.Lock()
	defer c.rateMu.Unlock()

	if c.rateTicker != nil {
		c.rateTicker.Stop()
		c.rateTicker = nil
		c.ratePrimed = false
	}

	if rps < minRatePerSecond {
		return
	}

	interval := time.Second / time.Duration(rps)
	if interval <= 0 {
		interval = time.Second
	}

	c.rateTicker = time.NewTicker(interval)
	c.ratePrimed = false
}

func (c *Client) waitForRateSlot() {
	c.rateMu.Lock()
	ticker := c.rateTicker
	primed := c.ratePrimed
	if !primed {
		c.ratePrimed = true
		c.rateMu.Unlock()
		return
	}
	c.rateMu.Unlock()

	if ticker == nil {
		return
	}

	<-ticker.C
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

type APIError struct {
	Status     int
	Body       string
	retryAfter time.Duration
}

func newAPIError(status int, body []byte, retryHeader string) *APIError {
	return &APIError{
		Status:     status,
		Body:       strings.TrimSpace(string(body)),
		retryAfter: parseRetryAfter(retryHeader),
	}
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API request failed with status %d: %s", e.Status, e.Body)
}

func (e *APIError) retryDelay() time.Duration {
	if e.retryAfter > 0 {
		return e.retryAfter
	}
	return 2 * time.Second
}

func getEnvOrFail(key string) string {
	value := os.Getenv(key)
	if value == "" {
		fmt.Fprintf(os.Stderr, "Error: %s environment variable is required\n", key)
		os.Exit(1)
	}
	return value
}

func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if until := time.Until(t); until > 0 {
			return until
		}
	}
	return 0
}
