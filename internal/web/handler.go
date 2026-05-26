package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/angoo/agentfoundry-ui/internal/api"
)

type Handler struct {
	client *api.Client
}

func NewHandler(backendURL string) (*Handler, error) {
	client, err := api.NewClient(backendURL)
	if err != nil {
		return nil, fmt.Errorf("create API client: %w", err)
	}

	return &Handler{client: client}, nil
}

func (h *Handler) Client() *api.Client {
	return h.client
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /chat/sessions/list", h.jsonSessionList)
	mux.HandleFunc("GET /chat/sessions/{id}", h.jsonSessionGet)
	mux.HandleFunc("POST /chat/sessions", h.jsonCreateSession)
	mux.HandleFunc("POST /chat/sessions/{id}/messages", h.jsonPostMessage)
	mux.HandleFunc("GET /chat/runs/{id}/events", h.runEvents)

	mux.HandleFunc("GET /agents/list", h.jsonAgentList)
	mux.HandleFunc("GET /agents/{name}/edit", h.jsonAgentGet)
	mux.HandleFunc("PUT /agents/{name}", h.jsonSaveAgent)
	mux.HandleFunc("POST /agents/form", h.jsonCreateAgent)
	mux.HandleFunc("POST /agents/{name}/clone", h.jsonCloneAgent)
	mux.HandleFunc("DELETE /agents/{name}", h.jsonDeleteAgent)

	mux.HandleFunc("GET /agents/{name}/versions", h.jsonListVersions)
	mux.HandleFunc("POST /agents/{name}/rollback", h.jsonRollback)

	mux.HandleFunc("GET /tools/servers/list", h.jsonMCPServerList)
	mux.HandleFunc("POST /tools/servers", h.jsonCreateMCPServer)
	mux.HandleFunc("GET /tools/servers/{name}", h.jsonGetMCPServer)
	mux.HandleFunc("PUT /tools/servers/{id}", h.jsonUpdateMCPServer)
	mux.HandleFunc("DELETE /tools/servers/{id}", h.jsonDeleteMCPServer)
	mux.HandleFunc("PUT /tools/servers/{id}/tools/{tool}", h.jsonSetToolScope)
	mux.HandleFunc("POST /tools/servers/{id}/refresh", h.jsonRefreshMCPServer)

	mux.HandleFunc("GET /api/keys", h.jsonAPIKeysList)
	mux.HandleFunc("POST /api/keys", h.jsonCreateAPIKey)
	mux.HandleFunc("DELETE /api/keys/{id}", h.jsonRevokeAPIKey)

	mux.HandleFunc("/api/v1/", h.proxyBackend)

	slog.Info("web UI routes registered")
}

func (h *Handler) jsonSessionList(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.client.ListSessions(r.Context())
	if err != nil {
		slog.Error("failed to list sessions", "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}
	writeJSON(w, sessions)
}

func (h *Handler) jsonSessionGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := h.client.GetSession(r.Context(), id)
	if err != nil {
		slog.Error("failed to get session", "id", id, "error", err)
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	writeJSON(w, sess)
}

func (h *Handler) jsonCreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: agent_id required", http.StatusBadRequest)
		return
	}
	if req.AgentID == "" {
		http.Error(w, "agent_id is required", http.StatusBadRequest)
		return
	}

	sess, err := h.client.CreateSession(r.Context(), req.AgentID)
	if err != nil {
		slog.Error("failed to create session", "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}
	writeJSON(w, sess)
}

func (h *Handler) jsonPostMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: message required", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	result, err := h.client.PostMessage(r.Context(), sessionID, req.Message)
	if err != nil {
		slog.Error("failed to post message", "session", sessionID, "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}
	writeJSON(w, result)
}

func (h *Handler) runEvents(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	reader, err := h.client.StreamRunEventsReader(r.Context(), runID)
	if err != nil {
		slog.Error("failed to connect to SSE stream", "run", runID, "error", err)
		http.Error(w, "stream error", http.StatusBadGateway)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
			flusher.Flush()
		}
		if err != nil {
			if err != io.EOF {
				slog.Error("SSE read error", "run", runID, "error", err)
			}
			return
		}
	}
}

func (h *Handler) jsonAgentList(w http.ResponseWriter, r *http.Request) {
	agents, err := h.client.ListAgents(r.Context())
	if err != nil {
		slog.Error("failed to list agents", "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}
	writeJSON(w, agents)
}

func (h *Handler) jsonAgentGet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	def, err := h.client.GetAgent(r.Context(), name)
	if err != nil {
		slog.Error("failed to get agent", "name", name, "error", err)
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}
	writeJSON(w, def)
}

func (h *Handler) jsonSaveAgent(w http.ResponseWriter, r *http.Request) {
	originalName := r.PathValue("name")
	def, err := definitionFromJSON(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := def.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	saved, err := h.client.UpdateAgent(r.Context(), originalName, def)
	if err != nil {
		slog.Error("failed to save agent", "name", def.Name, "error", err)
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}
	writeJSON(w, saved)
}

func (h *Handler) jsonCreateAgent(w http.ResponseWriter, r *http.Request) {
	def, err := definitionFromJSON(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := def.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	saved, err := h.client.CreateAgent(r.Context(), def)
	if err != nil {
		slog.Error("failed to create agent", "name", def.Name, "error", err)
		http.Error(w, "failed to create", http.StatusInternalServerError)
		return
	}
	writeJSON(w, saved)
}

func (h *Handler) jsonCloneAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	src, err := h.client.GetAgent(r.Context(), name)
	if err != nil {
		http.Error(w, "agent not found: "+name, http.StatusNotFound)
		return
	}

	agents, _ := h.client.ListAgents(r.Context())
	existing := make(map[string]bool, len(agents))
	for _, a := range agents {
		existing[a.Name] = true
	}

	cloneName, err := cloneAgentName(name, func(s string) bool { return existing[s] })
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	clone := *src
	clone.Name = cloneName
	clone.AgentID = ""
	clone.Tools = append([]string(nil), src.Tools...)
	if src.StructuredOutput != nil {
		so := *src.StructuredOutput
		clone.StructuredOutput = &so
	}

	saved, err := h.client.CreateAgent(r.Context(), &clone)
	if err != nil {
		slog.Error("failed to clone agent", "source", name, "clone", cloneName, "error", err)
		http.Error(w, "failed to clone", http.StatusInternalServerError)
		return
	}
	writeJSON(w, saved)
}

func (h *Handler) jsonDeleteAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := h.client.DeleteAgent(r.Context(), name); err != nil {
		slog.Error("failed to delete agent", "name", name, "error", err)
		http.Error(w, "failed to delete", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) jsonListVersions(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	versions, err := h.client.ListVersions(r.Context(), name)
	if err != nil {
		slog.Error("failed to list versions", "name", name, "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}
	writeJSON(w, versions)
}

func (h *Handler) jsonRollback(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	versionID := r.URL.Query().Get("version_id")
	if versionID == "" {
		http.Error(w, "version_id query parameter required", http.StatusBadRequest)
		return
	}
	def, err := h.client.Rollback(r.Context(), name, versionID)
	if err != nil {
		slog.Error("failed to rollback", "name", name, "version", versionID, "error", err)
		http.Error(w, "rollback failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, def)
}

func (h *Handler) jsonMCPServerList(w http.ResponseWriter, r *http.Request) {
	servers, err := h.client.ListMCPServers(r.Context())
	if err != nil {
		slog.Error("failed to list mcp servers", "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}
	writeJSON(w, servers)
}

func (h *Handler) jsonCreateMCPServer(w http.ResponseWriter, r *http.Request) {
	var req api.CreateMCPServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.URL == "" {
		http.Error(w, "name and url are required", http.StatusBadRequest)
		return
	}
	if req.Transport == "" {
		req.Transport = "sse"
	}
	if req.Scope == "" {
		req.Scope = "user"
	}

	server, err := h.client.CreateMCPServer(r.Context(), req)
	if err != nil {
		slog.Error("failed to create mcp server", "name", req.Name, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, server)
}

func (h *Handler) jsonGetMCPServer(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	server, err := h.client.GetMCPServer(r.Context(), name)
	if err != nil {
		slog.Error("failed to get mcp server", "name", name, "error", err)
		http.Error(w, "mcp server not found", http.StatusNotFound)
		return
	}
	writeJSON(w, server)
}

func (h *Handler) jsonUpdateMCPServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req api.CreateMCPServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	server, err := h.client.UpdateMCPServer(r.Context(), id, req)
	if err != nil {
		slog.Error("failed to update mcp server", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, server)
}

func (h *Handler) jsonDeleteMCPServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.client.DeleteMCPServer(r.Context(), id); err != nil {
		slog.Error("failed to delete mcp server", "id", id, "error", err)
		http.Error(w, "failed to delete", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) jsonSetToolScope(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("id")
	toolName := r.PathValue("tool")
	var req api.SetToolScopeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.client.SetToolScope(r.Context(), serverID, toolName, req); err != nil {
		slog.Error("failed to set tool scope", "server", serverID, "tool", toolName, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (h *Handler) jsonRefreshMCPServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.client.RefreshMCPServer(r.Context(), id); err != nil {
		slog.Error("failed to refresh mcp server", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (h *Handler) jsonAPIKeysList(w http.ResponseWriter, r *http.Request) {
	keys, err := h.client.ListAPIKeys(r.Context())
	if err != nil {
		slog.Error("failed to list api keys", "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}
	writeJSON(w, keys)
}

func (h *Handler) jsonCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	key, err := h.client.CreateAPIKey(r.Context(), req.Name)
	if err != nil {
		slog.Error("failed to create api key", "error", err)
		http.Error(w, "failed to create key", http.StatusInternalServerError)
		return
	}
	writeJSON(w, key)
}

func (h *Handler) jsonRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.client.RevokeAPIKey(r.Context(), id); err != nil {
		slog.Error("failed to revoke api key", "id", id, "error", err)
		http.Error(w, "failed to revoke", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) proxyBackend(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("failed to read proxy request body", "error", err)
		http.Error(w, "proxy error", http.StatusBadGateway)
		return
	}
	defer r.Body.Close()

	resp, err := h.client.Proxy(r.Context(), r.Method, r.URL.Path, r.URL.RawQuery, bytes.NewReader(body), r.Header.Get("Content-Type"))
	if err != nil {
		slog.Error("proxy request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		http.Error(w, "proxy error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)

	ct := resp.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "text/event-stream") {
		rc := http.NewResponseController(w)
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				w.Write(buf[:n])
				rc.Flush()
			}
			if err != nil {
				if err != io.EOF {
					slog.Error("proxy stream read error", "error", err)
				}
				return
			}
		}
	}

	io.Copy(w, resp.Body)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}

func definitionFromJSON(r *http.Request) (*api.Definition, error) {
	var formData struct {
		Kind               string                `json:"kind"`
		Name               string                `json:"name"`
		Description        string                `json:"description"`
		Model              string                `json:"model"`
		SystemPrompt       string                `json:"system_prompt"`
		Tools              []string              `json:"tools"`
		MaxTurns           int                   `json:"max_turns"`
		MaxConcurrentTools int                   `json:"max_concurrent_tools"`
		ForceJSON          bool                  `json:"force_json"`
		Scope              string                `json:"scope"`
		Team               string                `json:"team"`
		ProviderID         string                `json:"provider_id"`
		StructuredOutput   *api.StructuredOutput `json:"structured_output"`
	}
	if err := json.NewDecoder(r.Body).Decode(&formData); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	def := &api.Definition{
		Kind:               api.KindAgent,
		Name:               formData.Name,
		Description:        formData.Description,
		Model:              formData.Model,
		SystemPrompt:       formData.SystemPrompt,
		Tools:              formData.Tools,
		MaxTurns:           formData.MaxTurns,
		MaxConcurrentTools: formData.MaxConcurrentTools,
		ForceJSON:          formData.ForceJSON,
		Scope:              formData.Scope,
		Team:               formData.Team,
		ProviderID:         formData.ProviderID,
		StructuredOutput:   formData.StructuredOutput,
	}
	return def, nil
}

func cloneAgentName(src string, exists func(string) bool) (string, error) {
	candidate := src + "-copy"
	if !exists(candidate) {
		return candidate, nil
	}
	for i := 2; i <= 10; i++ {
		candidate = fmt.Sprintf("%s-copy-%d", src, i)
		if !exists(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("too many copies of %q", src)
}
