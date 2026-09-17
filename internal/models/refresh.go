package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

// modelListResponse mirrors the upstream /api/ide/v1/model_list payload
// (verified against a captured live response).
type modelListResponse struct {
	ModelConfigs []struct {
		Name            string `json:"name"`
		DisplayName     string `json:"display_name"`
		IsPreset        bool   `json:"is_preset"`
		Status          bool   `json:"status"`
		PromptMaxTokens int    `json:"prompt_max_tokens"`
	} `json:"model_configs"`
}

// inferProvider guesses the provider from a remote model id.
func inferProvider(id string) string {
	n := Normalize(id)
	switch {
	case strings.HasPrefix(n, "deepseek"), strings.HasPrefix(n, "ds-"):
		return "DeepSeek"
	case strings.HasPrefix(n, "glm"), strings.HasPrefix(n, "z-ai"):
		return "Zhipu"
	case strings.HasPrefix(n, "qwen"):
		return "Alibaba"
	case strings.HasPrefix(n, "kimi"):
		return "Moonshot"
	case strings.HasPrefix(n, "minimax"):
		return "MiniMax"
	case strings.HasPrefix(n, "seed"), strings.HasPrefix(n, "doubao"):
		return "ByteDance"
	case strings.HasPrefix(n, "gpt"):
		return "OpenAI"
	case strings.HasPrefix(n, "gemini"):
		return "Google"
	case strings.HasPrefix(n, "claude"):
		return "Anthropic"
	default:
		return "Unknown"
	}
}

// RefreshFromModelList merges the live model_list response into the registry.
// Only preset models (is_preset && status) are usable through llm_raw_chat —
// custom/connect entries (client_connect or ak-bearing) answer 4023
// MODEL_NOT_EXISTED and are skipped.
func (r *Registry) RefreshFromModelList(body []byte) (added, updated int, err error) {
	var resp modelListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, fmt.Errorf("failed to parse model_list response: %w", err)
	}

	for _, cfg := range resp.ModelConfigs {
		if cfg.Name == "" || !cfg.IsPreset || !cfg.Status {
			continue
		}
		display := firstNonEmpty(cfg.DisplayName, cfg.Name)
		provider := inferProvider(cfg.Name)

		if existing := r.Get(cfg.Name); existing != nil {
			if existing.MaxTokens != cfg.PromptMaxTokens && cfg.PromptMaxTokens > 0 {
				updated++
			}
			r.Upsert(ModelInfo{
				ModelID:       existing.ModelID,
				DisplayName:   firstNonEmpty(display, existing.DisplayName),
				Provider:      existing.Provider,
				MaxTokens:     cfg.PromptMaxTokens,
				ContextWindow: existing.ContextWindow,
				Source:        existing.Source,
				UpstreamID:    cfg.Name,
			})
			continue
		}

		r.Upsert(ModelInfo{
			ModelID:    cfg.Name,
			DisplayName: display,
			Provider:   provider,
			MaxTokens:  cfg.PromptMaxTokens,
			Source:     "remote",
			UpstreamID: cfg.Name,
			Aliases:    []string{display},
		})
		added++
	}

	r.linkUpstreamByDisplayName()
	return added, updated, nil
}

// linkUpstreamByDisplayName connects builtin client-facing models to live
// upstream model names when their display names match (normalized).
func (r *Registry) linkUpstreamByDisplayName() {
	r.mu.Lock()
	defer r.mu.Unlock()

	norm := func(s string) string {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "-", "")
		return s
	}

	var remotes []*ModelInfo
	for _, m := range r.models {
		if m.Source == "remote" || m.UpstreamID != "" {
			remotes = append(remotes, m)
		}
	}
	for _, b := range r.models {
		if b.Source != "builtin" || b.UpstreamID != "" {
			continue
		}
		for _, rm := range remotes {
			if norm(rm.DisplayName) != "" && norm(rm.DisplayName) == norm(b.ModelID) {
				b.UpstreamID = rm.UpstreamID
				break
			}
			if norm(rm.DisplayName) != "" && norm(rm.DisplayName) == norm(b.DisplayName) {
				b.UpstreamID = rm.UpstreamID
				break
			}
		}
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
