package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
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

	mux.HandleFunc("GET /tools/list", h.jsonToolList)
	mux.HandleFunc("POST /tools/generate", h.jsonToolGenerate)

	mux.HandleFunc("GET /api/keys", h.jsonAPIKeysList)
	mux.HandleFunc("POST /api/keys", h.jsonCreateAPIKey)
	mux.HandleFunc("DELETE /api/keys/{id}", h.jsonRevokeAPIKey)

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
		AgentName string `json:"agent_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: agent_name required", http.StatusBadRequest)
		return
	}
	if req.AgentName == "" {
		http.Error(w, "agent_name is required", http.StatusBadRequest)
		return
	}

	sess, err := h.client.CreateSession(r.Context(), req.AgentName)
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

func (h *Handler) jsonToolList(w http.ResponseWriter, r *http.Request) {
	servers, err := h.buildServerTools(r.Context())
	if err != nil {
		slog.Error("failed to list tools", "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}
	writeJSON(w, servers)
}

func (h *Handler) jsonToolGenerate(w http.ResponseWriter, r *http.Request) {
	var tools []string
	if err := json.NewDecoder(r.Body).Decode(&tools); err != nil {
		http.Error(w, "invalid JSON: expected string array", http.StatusBadRequest)
		return
	}
	sort.Strings(tools)

	var buf strings.Builder
	if len(tools) > 0 {
		buf.WriteString("tools:\n")
		for _, t := range tools {
			buf.WriteString("  - ")
			buf.WriteString(t)
			buf.WriteString("\n")
		}
	}

	writeJSON(w, map[string]any{
		"yaml":     buf.String(),
		"selected": len(tools),
		"lines":    len(tools) + 1,
	})
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

func (h *Handler) buildServerTools(ctx context.Context) ([]serverTools, error) {
	allTools, err := h.client.ListTools(ctx)
	if err != nil {
		return nil, err
	}

	byServer := make(map[string][]toolInfo)
	for _, t := range allTools {
		byServer[t.Server] = append(byServer[t.Server], toolInfo{
			QualifiedName: t.QualifiedName,
			Server:        t.Server,
			Name:          t.Name,
			Description:   t.Description,
		})
	}

	servers := make([]serverTools, 0, len(byServer))
	for srv, tools := range byServer {
		servers = append(servers, serverTools{Name: srv, Tools: tools})
	}
	sort.Slice(servers, func(i, j int) bool { return servers[i].Name < servers[j].Name })
	return servers, nil
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
		StructuredOutput:   formData.StructuredOutput,
	}
	return def, nil
}

type toolInfo struct {
	QualifiedName string `json:"qualified_name"`
	Server        string `json:"server"`
	Name          string `json:"name"`
	Description   string `json:"description"`
}

type serverTools struct {
	Name  string     `json:"name"`
	Tools []toolInfo `json:"tools"`
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
