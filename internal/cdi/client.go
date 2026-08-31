package cdi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

const bcbLastCDI = "https://api.bcb.gov.br/dados/serie/bcdata.sgs.12/dados/ultimos/1?formato=json"

// Client fetches the CDI annual rate from the Banco Central SGS.
type Client struct {
	httpClient *http.Client
	mu         sync.Mutex
	cached     float64
	cachedAt   time.Time
	ttl        time.Duration
}

// NewClient creates a CDI client with a 12h cache.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 8 * time.Second},
		ttl:        12 * time.Hour,
	}
}

// AnnualRate returns CDI in % a.a. Cached values are reused until ttl expires.
func (c *Client) AnnualRate(ctx context.Context) (float64, error) {
	c.mu.Lock()
	if c.cached > 0 && time.Since(c.cachedAt) < c.ttl {
		rate := c.cached
		c.mu.Unlock()
		return rate, nil
	}
	c.mu.Unlock()

	rate, err := c.fetch(ctx)
	if err != nil {
		c.mu.Lock()
		fallback := c.cached
		c.mu.Unlock()
		if fallback > 0 {
			return fallback, nil
		}
		return 0, err
	}

	c.mu.Lock()
	c.cached = rate
	c.cachedAt = time.Now()
	c.mu.Unlock()
	return rate, nil
}

func (c *Client) fetch(ctx context.Context) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bcbLastCDI, nil)
	if err != nil {
		return 0, fmt.Errorf("cdi request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("cdi fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("cdi status %d", resp.StatusCode)
	}

	var rows []struct {
		Valor string `json:"valor"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return 0, fmt.Errorf("cdi decode: %w", err)
	}
	if len(rows) == 0 {
		return 0, fmt.Errorf("cdi empty series")
	}
	raw, err := strconv.ParseFloat(rows[0].Valor, 64)
	if err != nil {
		return 0, fmt.Errorf("cdi value: %w", err)
	}
	annual := models.CDIDailyToAnnual(raw)
	if annual <= 0 {
		return 0, fmt.Errorf("cdi invalid rate %v", raw)
	}
	return annual, nil
}
