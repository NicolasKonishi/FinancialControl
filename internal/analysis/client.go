package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

// Client calls the Python financial analysis service over HTTP/JSON.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates an analysis Client with a request timeout.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// MonthlyAnalysis posts transactions to Python and returns the computed summary.
func (c *Client) MonthlyAnalysis(ctx context.Context, req models.MonthlyAnalysisRequest) (models.MonthlyAnalysisResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return models.MonthlyAnalysisResponse{}, fmt.Errorf("marshal analysis request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/analysis/monthly", bytes.NewReader(body))
	if err != nil {
		return models.MonthlyAnalysisResponse{}, fmt.Errorf("create analysis request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return models.MonthlyAnalysisResponse{}, fmt.Errorf("call analysis service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.MonthlyAnalysisResponse{}, fmt.Errorf("analysis service returned status %d", resp.StatusCode)
	}

	var result models.MonthlyAnalysisResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return models.MonthlyAnalysisResponse{}, fmt.Errorf("decode analysis response: %w", err)
	}
	return result, nil
}
