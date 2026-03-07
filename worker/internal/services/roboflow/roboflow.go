package roboflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"govision/worker/internal/domain"
)

const defaultTimeout = 120 * time.Second

// APIError represents a non-retryable HTTP error from the Roboflow API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("roboflow returned status %d: %s", e.StatusCode, e.Body)
}

type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (c *Client) Detect(ctx context.Context, imageURL string) (*domain.RoboflowResponse, error) {
	log.Printf("[ROBOFLOW] - Sending image URL to Roboflow: %s", imageURL)

	result, err := c.infer(ctx, imageURL)
	if err != nil {
		return nil, fmt.Errorf("roboflow inference failed: %w", err)
	}

	predCount := len(result.Predictions)

	log.Printf("[ROBOFLOW] - Inference completed. %d prediction(s) returned.", predCount)
	return result, nil
}

func (c *Client) infer(ctx context.Context, imageURL string) (*domain.RoboflowResponse, error) {
	endpoint := fmt.Sprintf(
		"https://serverless.roboflow.com/%s?api_key=%s&image=%s",
		c.model,
		c.apiKey,
		url.QueryEscape(imageURL),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to roboflow failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(respBytes)}
	}

	var roboflowResp domain.RoboflowResponse
	if err := json.Unmarshal(respBytes, &roboflowResp); err != nil {
		return nil, fmt.Errorf("failed to decode roboflow response: %w", err)
	}

	fmt.Printf("RESULT RAW: %+v", roboflowResp)

	return &roboflowResp, nil
}
