package lingo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	DefaultBaseURL = "https://openrouter.ai/api/v1"
	DefaultModel   = "google/gemma-4-26b-a4b-it:free"
	DefaultKeyEnv  = "README_LINGO_API_KEY"
)

type ClientConfig struct {
	BaseURL string
	Model   string
	APIKey  string
	Client  *http.Client
}

type Client struct {
	baseURL string
	model   string
	apiKey  string
	client  *http.Client
}

type TranslateRequest struct {
	SourcePath string
	Target     string
	Markdown   string
	Glossary   string
}

type Translator interface {
	Translate(context.Context, TranslateRequest) (string, error)
}

func NewClient(config ClientConfig) *Client {
	if config.BaseURL == "" {
		config.BaseURL = DefaultBaseURL
	}
	if config.Model == "" {
		config.Model = DefaultModel
	}
	if config.Client == nil {
		config.Client = http.DefaultClient
	}
	return &Client{
		baseURL: strings.TrimRight(config.BaseURL, "/"),
		model:   config.Model,
		apiKey:  config.APIKey,
		client:  config.Client,
	}
}

func (c *Client) Translate(ctx context.Context, req TranslateRequest) (string, error) {
	if c.apiKey == "" {
		return "", errors.New("API key is required")
	}
	userPrompt := fmt.Sprintf("Translate this Markdown file (%s) into %s. Return only the translated Markdown.", req.SourcePath, req.Target)
	if strings.TrimSpace(req.Glossary) != "" {
		userPrompt += fmt.Sprintf("\n\nUse this project glossary and terminology guidance while translating. Preserve named terms as instructed:\n\n%s", req.Glossary)
	}
	userPrompt += "\n\n" + req.Markdown
	body := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: "You translate Markdown documentation; preserve Markdown structure, code fences, links, tables, front matter, and HTML comments. Translate prose only.",
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		Temperature: 0.2,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("chat completions request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("chat completions request failed with status %s: %s", resp.Status, string(respBody))
	}
	var decoded chatResponse
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return "", fmt.Errorf("decode chat completions response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return "", errors.New("chat completions response had no choices")
	}
	return decoded.Choices[0].Message.Content, nil
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}
