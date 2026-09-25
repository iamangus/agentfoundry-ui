package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Session struct {
	ID          string    `json:"id"`
	AgentID     string    `json:"agent_id"`
	AgentName   string    `json:"agent_name"`
	Messages    []Message `json:"messages"`
	CreatedAt   time.Time `json:"created_at"`
	ActiveRunID string    `json:"active_run_id,omitempty"`
}

type Message struct {
	Role    string    `json:"role"`
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
}

type APIKeyInfo struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	FullKey    string     `json:"full_key"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

type MCPServerInfo struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	URL           string            `json:"url"`
	Transport     string            `json:"transport"`
	Headers       map[string]string `json:"headers,omitempty"`
	Scope         string            `json:"scope"`
	Team          string            `json:"team,omitempty"`
	CreatedBy     string            `json:"created_by"`
	CreatedAt     string            `json:"created_at"`
	UpdatedAt     string            `json:"updated_at"`
	Connected     bool              `json:"connected"`
	Tools         []MCPServerTool   `json:"tools"`
	ToolOverrides json.RawMessage   `json:"tool_overrides,omitempty"`
}

type MCPServerTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Scope       string `json:"scope"`
	ScopeSource string `json:"scope_source"`
}

type CreateMCPServerRequest struct {
	Name          string            `json:"name"`
	URL           string            `json:"url"`
	Transport     string            `json:"transport"`
	Headers       map[string]string `json:"headers,omitempty"`
	Scope         string            `json:"scope"`
	Team          string            `json:"team,omitempty"`
	ToolOverrides json.RawMessage   `json:"tool_overrides,omitempty"`
}

type SetToolScopeRequest struct {
	Scope string `json:"scope"`
	Team  string `json:"team,omitempty"`
}

type TokenProvider interface {
	GetAccessToken(ctx context.Context) string
}

type Client struct {
	baseURL       *url.URL
	httpClient    *http.Client
	tokenProvider TokenProvider
}

func NewClient(backendURL string) (*Client, error) {
	u, err := url.Parse(backendURL)
	if err != nil {
		return nil, fmt.Errorf("parse backend URL: %w", err)
	}
	return &Client{
		baseURL:    u,
		httpClient: &http.Client{Timeout: 0},
	}, nil
}

func (c *Client) SetTokenProvider(tp TokenProvider) {
	c.tokenProvider = tp
}

func (c *Client) withAuth(req *http.Request) {
	if c.tokenProvider != nil {
		if token := c.tokenProvider.GetAccessToken(req.Context()); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
}

func (c *Client) get(ctx context.Context, path string) (*http.Response, error) {
	u := c.baseURL.JoinPath(path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	c.withAuth(req)
	return c.httpClient.Do(req)
}

func (c *Client) postJSON(ctx context.Context, path string, body any) (*http.Response, error) {
	u := c.baseURL.JoinPath(path)
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.withAuth(req)
	return c.httpClient.Do(req)
}

func (c *Client) putJSON(ctx context.Context, path string, body any) (*http.Response, error) {
	u := c.baseURL.JoinPath(path)
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.withAuth(req)
	return c.httpClient.Do(req)
}

func (c *Client) delete(ctx context.Context, path string) (*http.Response, error) {
	u := c.baseURL.JoinPath(path)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), nil)
	if err != nil {
		return nil, err
	}
	c.withAuth(req)
	return c.httpClient.Do(req)
}

func (c *Client) decodeJSON(resp *http.Response, v any) error {
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		body, _ := io.ReadAll(resp.Body)
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("%s: %s", resp.Status, errResp.Error)
		}
		return fmt.Errorf("%s: %s", resp.Status, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func (c *Client) ListAgents(ctx context.Context) ([]*Definition, error) {
	resp, err := c.get(ctx, "/api/v1/agents")
	if err != nil {
		return nil, err
	}
	var agents []*Definition
	if err := c.decodeJSON(resp, &agents); err != nil {
		return nil, err
	}
	return agents, nil
}

func (c *Client) GetAgent(ctx context.Context, name string) (*Definition, error) {
	resp, err := c.get(ctx, "/api/v1/agents/"+url.PathEscape(name))
	if err != nil {
		return nil, err
	}
	var def Definition
	if err := c.decodeJSON(resp, &def); err != nil {
		return nil, err
	}
	return &def, nil
}

func (c *Client) CreateAgent(ctx context.Context, def *Definition) (*Definition, error) {
	resp, err := c.postJSON(ctx, "/api/v1/agents", def)
	if err != nil {
		return nil, err
	}
	var created Definition
	if err := c.decodeJSON(resp, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdateAgent(ctx context.Context, name string, def *Definition) (*Definition, error) {
	resp, err := c.putJSON(ctx, "/api/v1/agents/"+url.PathEscape(name), def)
	if err != nil {
		return nil, err
	}
	var updated Definition
	if err := c.decodeJSON(resp, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteAgent(ctx context.Context, name string) error {
	resp, err := c.delete(ctx, "/api/v1/agents/"+url.PathEscape(name))
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("delete agent: %s", resp.Status)
	}
	return nil
}

func (c *Client) ListVersions(ctx context.Context, name string) (*VersionsResponse, error) {
	resp, err := c.get(ctx, "/api/v1/agents/"+url.PathEscape(name)+"/versions")
	if err != nil {
		return nil, err
	}
	var versions VersionsResponse
	if err := c.decodeJSON(resp, &versions); err != nil {
		return nil, err
	}
	return &versions, nil
}

func (c *Client) Rollback(ctx context.Context, name, versionID string) (*Definition, error) {
	u := c.baseURL.JoinPath("/api/v1/agents/" + url.PathEscape(name) + "/rollback")
	u.RawQuery = "version_id=" + url.QueryEscape(versionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return nil, err
	}
	c.withAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	var def Definition
	if err := c.decodeJSON(resp, &def); err != nil {
		return nil, err
	}
	return &def, nil
}

func (c *Client) CreateSession(ctx context.Context, agentID string) (*Session, error) {
	resp, err := c.postJSON(ctx, "/api/v1/chat/sessions", map[string]string{"agent_id": agentID})
	if err != nil {
		return nil, err
	}
	var sess Session
	if err := c.decodeJSON(resp, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (c *Client) GetSession(ctx context.Context, id string) (*Session, error) {
	resp, err := c.get(ctx, "/api/v1/chat/sessions/"+url.PathEscape(id))
	if err != nil {
		return nil, err
	}
	var sess Session
	if err := c.decodeJSON(resp, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (c *Client) ListSessions(ctx context.Context) ([]*Session, error) {
	resp, err := c.get(ctx, "/api/v1/chat/sessions")
	if err != nil {
		return nil, err
	}
	var sessions []*Session
	if err := c.decodeJSON(resp, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (c *Client) StreamRunEvents(ctx context.Context, runID string) (*http.Response, error) {
	u := c.baseURL.JoinPath("/api/v1/runs/" + url.PathEscape(runID) + "/events")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	c.withAuth(req)
	return c.httpClient.Do(req)
}

func (c *Client) StreamRunEventsReader(ctx context.Context, runID string) (io.ReadCloser, error) {
	resp, err := c.StreamRunEvents(ctx, runID)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("SSE stream: %s: %s", resp.Status, string(body))
	}
	return resp.Body, nil
}

func (c *Client) CreateAPIKey(ctx context.Context, name string) (*APIKeyInfo, error) {
	resp, err := c.postJSON(ctx, "/api/v1/api-keys", map[string]string{"name": name})
	if err != nil {
		return nil, err
	}
	var key APIKeyInfo
	if err := c.decodeJSON(resp, &key); err != nil {
		return nil, err
	}
	return &key, nil
}

func (c *Client) ListAPIKeys(ctx context.Context) ([]APIKeyInfo, error) {
	resp, err := c.get(ctx, "/api/v1/api-keys")
	if err != nil {
		return nil, err
	}
	var keys []APIKeyInfo
	if err := c.decodeJSON(resp, &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

func (c *Client) RevokeAPIKey(ctx context.Context, id string) error {
	resp, err := c.delete(ctx, "/api/v1/api-keys/"+url.PathEscape(id))
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode != 404 && resp.StatusCode != 410 {
		return fmt.Errorf("revoke api key: %s", resp.Status)
	}
	return nil
}

func (c *Client) ListMCPServers(ctx context.Context) ([]MCPServerInfo, error) {
	resp, err := c.get(ctx, "/api/v1/mcp-servers")
	if err != nil {
		return nil, err
	}
	var servers []MCPServerInfo
	if err := c.decodeJSON(resp, &servers); err != nil {
		return nil, err
	}
	return servers, nil
}

func (c *Client) CreateMCPServer(ctx context.Context, req CreateMCPServerRequest) (*MCPServerInfo, error) {
	resp, err := c.postJSON(ctx, "/api/v1/mcp-servers", req)
	if err != nil {
		return nil, err
	}
	var server MCPServerInfo
	if err := c.decodeJSON(resp, &server); err != nil {
		return nil, err
	}
	return &server, nil
}

func (c *Client) GetMCPServer(ctx context.Context, name string) (*MCPServerInfo, error) {
	resp, err := c.get(ctx, "/api/v1/mcp-servers/"+url.PathEscape(name))
	if err != nil {
		return nil, err
	}
	var server MCPServerInfo
	if err := c.decodeJSON(resp, &server); err != nil {
		return nil, err
	}
	return &server, nil
}

func (c *Client) UpdateMCPServer(ctx context.Context, id string, req CreateMCPServerRequest) (*MCPServerInfo, error) {
	resp, err := c.putJSON(ctx, "/api/v1/mcp-servers/"+url.PathEscape(id), req)
	if err != nil {
		return nil, err
	}
	var server MCPServerInfo
	if err := c.decodeJSON(resp, &server); err != nil {
		return nil, err
	}
	return &server, nil
}

func (c *Client) DeleteMCPServer(ctx context.Context, id string) error {
	resp, err := c.delete(ctx, "/api/v1/mcp-servers/"+url.PathEscape(id))
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		return fmt.Errorf("delete mcp server: %s", resp.Status)
	}
	return nil
}

func (c *Client) SetToolScope(ctx context.Context, serverID, toolName string, req SetToolScopeRequest) error {
	resp, err := c.putJSON(ctx, "/api/v1/mcp-servers/"+url.PathEscape(serverID)+"/tools/"+url.PathEscape(toolName), req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("set tool scope: %s", resp.Status)
	}
	return nil
}

func (c *Client) RefreshMCPServer(ctx context.Context, id string) error {
	resp, err := c.postJSON(ctx, "/api/v1/mcp-servers/"+url.PathEscape(id)+"/refresh", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("refresh mcp server: %s", resp.Status)
	}
	return nil
}

func (c *Client) Proxy(ctx context.Context, method, path, query string, body io.Reader, contentType string) (*http.Response, error) {
	u := c.baseURL.JoinPath(path)
	if query != "" {
		u.RawQuery = query
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	c.withAuth(req)
	return c.httpClient.Do(req)
}
