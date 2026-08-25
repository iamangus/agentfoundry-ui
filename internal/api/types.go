package api

import (
	"encoding/json"
	"errors"
)

type Kind string

const (
	KindAgent Kind = "agent"
)

var (
	ErrMissingName         = errors.New("definition: name is required")
	ErrMissingKind         = errors.New("definition: kind is required")
	ErrInvalidKind         = errors.New("definition: kind must be 'agent'")
	ErrMissingSystemPrompt = errors.New("definition: system_prompt is required")
)

type Definition struct {
	AgentID                string                  `json:"agent_id,omitempty"`
	ProviderID             string                  `json:"provider_id,omitempty"`
	Kind                   Kind                    `json:"kind"`
	Name                   string                  `json:"name"`
	Description            string                  `json:"description,omitempty"`
	Model                  string                  `json:"model,omitempty"`
	SystemPrompt           string                  `json:"system_prompt"`
	Tools                  []string                `json:"tools,omitempty"`
	MaxTurns               int                     `json:"max_turns,omitempty"`
	MaxConcurrentTools     int                     `json:"max_concurrent_tools,omitempty"`
	ForceJSON              bool                    `json:"force_json,omitempty"`
	StructuredOutput       *StructuredOutput       `json:"structured_output,omitempty"`
	Scope                  string                  `json:"scope,omitempty"`
	Team                   string                  `json:"team,omitempty"`
	CreatedBy              string                  `json:"created_by,omitempty"`
	MemoryEnabled          bool                    `json:"memory_enabled,omitempty"`
	MemorySearchAgentID    string                  `json:"memory_search_agent_id,omitempty"`
	MemoryIngestAgentID    string                  `json:"memory_ingest_agent_id,omitempty"`
	ToolOverrides          json.RawMessage         `json:"tool_overrides,omitempty"`
	ModelParams            json.RawMessage         `json:"model_params,omitempty"`
	PreInferenceProcessors []PreInferenceProcessor `json:"pre_inference_processors,omitempty"`
	HandoffTo              string                  `json:"handoff_to,omitempty"`
	Handoffs               []string                `json:"handoffs,omitempty"`
}

type PreInferenceProcessor struct {
	ID        string          `json:"id,omitempty"`
	Processor string          `json:"processor"`
	Phase     string          `json:"phase"`
	Config    json.RawMessage `json:"config"`
	OnError   string          `json:"on_error,omitempty"`
	Timeout   int             `json:"timeout,omitempty"`
}

type StructuredOutput struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema,omitempty"`
	Strict bool            `json:"strict,omitempty"`
}

type AgentVersion struct {
	VersionID    string `json:"version_id"`
	LastModified string `json:"last_modified"`
	IsLatest     bool   `json:"is_latest"`
}

type VersionsResponse struct {
	Versions []AgentVersion `json:"versions"`
}

func (d *Definition) Validate() error {
	if d.Name == "" {
		return ErrMissingName
	}
	if d.Kind == "" {
		return ErrMissingKind
	}
	if d.Kind != KindAgent {
		return ErrInvalidKind
	}
	if d.SystemPrompt == "" {
		return ErrMissingSystemPrompt
	}
	return nil
}
