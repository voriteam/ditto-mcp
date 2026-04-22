package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type executeParams struct {
	Statement string         `json:"statement" jsonschema:"DQL statement to execute."`
	Args      map[string]any `json:"args,omitempty" jsonschema:"Named parameters for parameterized queries."`
}

type dittoClient struct {
	url    string
	apiKey string
	client *http.Client
}

func (c *dittoClient) execute(ctx context.Context, params executeParams) (*mcp.CallToolResult, any, error) {
	args := params.Args
	if args == nil {
		args = map[string]any{}
	}
	payload, err := json.Marshal(struct {
		Statement string         `json:"statement"`
		Args      map[string]any `json:"args"`
	}{params.Statement, args})
	if err != nil {
		return nil, nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(payload))
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("ditto request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("Ditto request failed (%d %s): %s", resp.StatusCode, resp.Status, body),
			}},
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(body)}},
	}, nil, nil
}

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("missing required environment variable: %s", name)
	}
	return v
}

func ptr[T any](v T) *T { return &v }

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8002"
	}

	ditto := &dittoClient{
		url:    requireEnv("DITTO_API_URL"),
		apiKey: requireEnv("DITTO_API_KEY"),
		client: &http.Client{Timeout: 30 * time.Second},
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "ditto-mcp",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name: "execute_query",
		Description: "Execute a DQL statement against the Ditto HTTP API. " +
			"Supports SELECT, INSERT, UPDATE, DELETE, and UPSERT.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr(true),
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in executeParams) (*mcp.CallToolResult, any, error) {
		return ditto.execute(ctx, in)
	})

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	commitSHA := os.Getenv("COMMIT_SHA")
	if commitSHA == "" {
		commitSHA = "unknown"
	}

	addr := ":" + port
	log.Printf("ditto-mcp listening on %s (commit=%s)", addr, commitSHA)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
