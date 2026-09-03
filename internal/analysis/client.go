package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

// ErrStatementParse is returned when Python rejects a statement file.
var ErrStatementParse = errors.New("statement parse failed")

// StatementParseError carries the Python error code (encrypted, no_text, ...).
type StatementParseError struct {
	Code   string
	Status int
}

func (e *StatementParseError) Error() string {
	if e.Code != "" {
		return e.Code
	}
	return fmt.Sprintf("status %d", e.Status)
}

func (e *StatementParseError) Unwrap() error {
	return ErrStatementParse
}

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

func (c *Client) parseClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
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

type pythonDetail struct {
	Detail json.RawMessage `json:"detail"`
}

// ParseStatement sends a statement PDF/CSV to Python and returns extracted lines.
func (c *Client) ParseStatement(ctx context.Context, file []byte, year, month int) (models.ParsedStatement, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/statements/parse", bytes.NewReader(file))
	if err != nil {
		return models.ParsedStatement{}, fmt.Errorf("create statement request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/pdf")
	query := httpReq.URL.Query()
	if year > 0 {
		query.Set("year", strconv.Itoa(year))
	}
	if month > 0 {
		query.Set("month", strconv.Itoa(month))
	}
	httpReq.URL.RawQuery = query.Encode()

	resp, err := c.parseClient().Do(httpReq)
	if err != nil {
		return models.ParsedStatement{}, fmt.Errorf("call analysis service: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return models.ParsedStatement{}, fmt.Errorf("read statement response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		code := strings.TrimSpace(string(body))
		var detail pythonDetail
		if json.Unmarshal(body, &detail) == nil && len(detail.Detail) > 0 {
			var asString string
			if json.Unmarshal(detail.Detail, &asString) == nil {
				code = asString
			}
		}
		return models.ParsedStatement{}, &StatementParseError{Code: code, Status: resp.StatusCode}
	}

	var result models.ParsedStatement
	if err := json.Unmarshal(body, &result); err != nil {
		return models.ParsedStatement{}, fmt.Errorf("decode statement response: %w", err)
	}
	return result, nil
}
