package roboflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"govision/worker/internal/domain"
)

const defaultTimeout = 30 * time.Second

// APIError represents a non-retryable HTTP error from the Roboflow API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("roboflow returned status %d: %s", e.StatusCode, e.Body)
}

// inferRequest represents the JSON body sent to the Roboflow Workflows API.
type inferRequest struct {
	APIKey string      `json:"api_key"`
	Inputs inferInputs `json:"inputs"`
}

type inferInputs struct {
	Image inferImage `json:"image"`
}

type inferImage struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Client struct {
	apiKey      string
	workspaceID string
	workflowID  string
	httpClient  *http.Client
}

func NewClient(apiKey, workspaceID, workflowID string) *Client {
	return &Client{
		apiKey:      apiKey,
		workspaceID: workspaceID,
		workflowID:  workflowID,
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

	log.Printf("[ROBOFLOW] - Inference completed. %d prediction(s) returned.", len(result.Predictions))
	return result, nil
}

func (c *Client) infer(ctx context.Context, imageURL string) (*domain.RoboflowResponse, error) {
	url := fmt.Sprintf(
		"https://detect.roboflow.com/infer/workflows/%s/%s",
		c.workspaceID,
		c.workflowID,
	)

	reqBody := inferRequest{
		APIKey: c.apiKey,
		Inputs: inferInputs{
			Image: inferImage{
				Type:  "url",
				Value: imageURL,
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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

	return &roboflowResp, nil
}
