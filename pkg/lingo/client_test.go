package lingo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestClientTranslateUsesChatCompletionsShape(t *testing.T) {
	fakeKey := "placeholder-" + strings.ReplaceAll(t.Name(), "/", "-")
	var requestPath string
	var authHeader string
	var payload map[string]any

	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requestPath = r.URL.Path
		authHeader = r.Header.Get("Authorization")
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"# 标题\n\n正文"}}]}`)),
		}, nil
	})

	client := NewClient(ClientConfig{
		BaseURL: "https://example.invalid/api/v1",
		Model:   "google/gemma-4-26b-a4b-it:free",
		APIKey:  fakeKey,
		Client:  &http.Client{Transport: transport},
	})

	got, err := client.Translate(context.Background(), TranslateRequest{
		SourcePath: "README.md",
		Target:     "zh",
		Markdown:   "# Title\n\nBody",
		Glossary:   "Keep README as README.\nDo not translate readme-lingo.",
	})
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}
	if got != "# 标题\n\n正文" {
		t.Fatalf("translation = %q", got)
	}
	if requestPath != "/api/v1/chat/completions" {
		t.Fatalf("path = %q, want /api/v1/chat/completions", requestPath)
	}
	if authHeader != "Bearer "+fakeKey {
		t.Fatalf("authorization header was not set correctly")
	}
	if payload["model"] != "google/gemma-4-26b-a4b-it:free" {
		t.Fatalf("model = %v", payload["model"])
	}
	messages, ok := payload["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages payload malformed: %#v", payload["messages"])
	}
	combined, _ := json.Marshal(messages)
	if !strings.Contains(string(combined), "preserve Markdown") ||
		!strings.Contains(string(combined), "zh") ||
		!strings.Contains(string(combined), "Keep README as README") {
		t.Fatalf("prompt did not include expected translation guidance: %s", combined)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
