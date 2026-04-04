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
	Kind               Kind              `json:"kind"`
	Name               string            `json:"name"`
	Description        string            `json:"description,omitempty"`
	Model              string            `json:"model,omitempty"`
	SystemPrompt       string            `json:"system_prompt"`
	Tools              []string          `json:"tools,omitempty"`
	MaxTurns           int               `json:"max_turns,omitempty"`
	MaxConcurrentTools int               `json:"max_concurrent_tools,omitempty"`
	ForceJSON          bool              `json:"force_json,omitempty"`
	StructuredOutput   *StructuredOutput `json:"structured_output,omitempty"`
	Scope              string            `json:"scope,omitempty"`
	Team               string            `json:"team,omitempty"`
	CreatedBy          string            `json:"created_by,omitempty"`
}

type StructuredOutput struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema,omitempty"`
	Strict bool            `json:"strict,omitempty"`
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
