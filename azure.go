package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	azureIMDSEndpoint = "http://169.254.169.254/metadata/identity/oauth2/token"
	azureAudience     = "api://AzureADTokenExchange"
)

var azureHTTP = &http.Client{Timeout: 10 * time.Second}

// azureIMDSToken fetches a managed identity token for the Tailscale token exchange.
func azureIMDSToken(ctx context.Context, clientID string) (string, error) {
	q := url.Values{}
	q.Set("api-version", "2018-02-01")
	q.Set("resource", azureAudience)
	if clientID != "" {
		q.Set("client_id", clientID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, azureIMDSEndpoint+"?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata", "true")

	resp, err := azureHTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("azure imds request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("azure imds read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("azure imds: unexpected status %d: %s", resp.StatusCode, body)
	}

	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("azure imds decode: %w", err)
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("azure imds: empty access_token")
	}
	return out.AccessToken, nil
}
