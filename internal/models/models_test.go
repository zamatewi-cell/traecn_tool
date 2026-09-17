package models

import (
	"os"
	"testing"
)

// requiredModels is the exact model list the registry must ship with.
var requiredModels = []struct {
	id       string
	provider string
}{
	{"Seed-Evolving", "ByteDance"},
	{"Seed-2.1-Pro-0915", "ByteDance"},
	{"Seed-2.1-Turbo", "ByteDance"},
	{"Seed-Code", "ByteDance"},
	{"GLM-5.3-Flash", "Zhipu"},
	{"GLM-5.3", "Zhipu"},
	{"GLM-5.2", "Zhipu"},
	{"DeepSeek-V4.1", "DeepSeek"},
	{"DeepSeek-V4-Flash", "DeepSeek"},
	{"DeepSeek-V4-Pro", "DeepSeek"},
	{"Kimi-K3", "Moonshot"},
	{"Kimi-K2.8-Preview", "Moonshot"},
	{"MiniMax-M3", "MiniMax"},
	{"Qwen3.8-Flash", "Alibaba"},
	{"Qwen3.8-Max", "Alibaba"},
	{"Qwen3.7-Plus", "Alibaba"},
}

func TestRegistry_BuiltinModels(t *testing.T) {
	r := NewRegistry()
	for _, want := range requiredModels {
		m := r.Get(want.id)
		if m == nil {
			t.Errorf("builtin model %q missing from registry", want.id)
			continue
		}
		if m.Provider != want.provider {
			t.Errorf("model %q provider = %q, want %q", want.id, m.Provider, want.provider)
		}
		if m.MaxTokens <= 0 || m.ContextWindow <= 0 {
			t.Errorf("model %q has invalid token metadata: %+v", want.id, m)
		}
	}
}

func TestRegistry_AliasResolution(t *testing.T) {
	r := NewRegistry()
	tests := []struct {
		input string
		want  string
	}{
		// Spec-mandated alias mappings.
		{"deepseek-v4", "DeepSeek-V4-Flash"},
		{"deepseek-chat", "DeepSeek-V4-Flash"},
		{"deepseek-r1", "DeepSeek-V4-Pro"},
		{"deepseek-reasoner", "DeepSeek-V4-Pro"},
		{"glm-5", "GLM-5.3"},
		{"glm", "GLM-5.3"},
		{"qwen", "Qwen3.7-Plus"},
		{"kimi", "Kimi-K3"},
		{"doubao", "Seed-Code"},
		{"seed", "Seed-Code"},
		// Case-insensitive exact IDs.
		{"deepseek-v4-flash", "DeepSeek-V4-Flash"},
		{"GLM-5.3", "GLM-5.3"},
		{"kimi-k3", "Kimi-K3"},
		{"QWEN3.8-MAX", "Qwen3.8-Max"},
		// Normalization: underscores and whitespace.
		{"deepseek_v4", "DeepSeek-V4-Flash"},
		{"  glm-5  ", "GLM-5.3"},
		// Unique prefix.
		{"glm-5.3-f", "GLM-5.3-Flash"},
		{"kimi-k2.8-p", "Kimi-K2.8-Preview"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := r.Resolve(tt.input)
			if got == nil {
				t.Fatalf("Resolve(%q) = nil, want %q", tt.input, tt.want)
			}
			if got.ModelID != tt.want {
				t.Errorf("Resolve(%q) = %q, want %q", tt.input, got.ModelID, tt.want)
			}
		})
	}
}

func TestRegistry_ResolveUnknown(t *testing.T) {
	r := NewRegistry()
	if got := r.Resolve("nonexistent-model-xyz"); got != nil {
		t.Errorf("Resolve(nonexistent) = %v, want nil", got)
	}
	if got := r.Resolve(""); got != nil {
		t.Errorf("Resolve(\"\") = %v, want nil", got)
	}
}

func TestRegistry_ListNoDuplicates(t *testing.T) {
	r := NewRegistry()
	seen := make(map[string]bool)
	for _, m := range r.List() {
		if seen[m.ModelID] {
			t.Errorf("duplicate model in List(): %q", m.ModelID)
		}
		seen[m.ModelID] = true
		if m.DisplayName == "" || m.Provider == "" {
			t.Errorf("model %q missing display name or provider", m.ModelID)
		}
	}
}

func TestRegistry_RefreshFromModelList(t *testing.T) {
	r := NewRegistry()
	// Real model_list shape: only is_preset && status entries are usable
	// through llm_raw_chat; custom/connect entries must be skipped.
	body := []byte(`{
	  "model_configs": [
	    {"name": "seed_m8", "display_name": "Doubao-1.5-pro", "is_preset": true, "status": true, "prompt_max_tokens": 28000},
	    {"name": "deepseek-V3", "display_name": "DeepSeek-V3", "is_preset": true, "status": true, "prompt_max_tokens": 40000},
	    {"name": "broken-preset", "display_name": "Broken", "is_preset": true, "status": false, "prompt_max_tokens": 1000},
	    {"name": "custom-thing", "display_name": "Custom", "client_connect": true, "status": true, "prompt_max_tokens": 30000}
	  ]
	}`)

	added, updated, err := r.RefreshFromModelList(body)
	if err != nil {
		t.Fatalf("RefreshFromModelList() error = %v", err)
	}
	if added != 2 || updated != 0 {
		t.Errorf("added=%d updated=%d, want 2/0", added, updated)
	}

	m := r.Get("seed_m8")
	if m == nil || m.Source != "remote" || m.UpstreamID != "seed_m8" || m.MaxTokens != 28000 {
		t.Errorf("seed_m8 = %+v", m)
	}
	if got := r.UpstreamFor("Doubao-1.5-pro"); got != "seed_m8" {
		t.Errorf("UpstreamFor(display alias) = %q, want seed_m8", got)
	}
	if r.Get("broken-preset") != nil {
		t.Error("status=false preset must be skipped")
	}
	if r.Get("custom-thing") != nil {
		t.Error("custom/connect model must be skipped (llm_raw_chat 4023s it)")
	}

	// Second refresh: metadata update path.
	body2 := []byte(`{"model_configs":[{"name":"seed_m8","display_name":"Doubao-1.5-pro","is_preset":true,"status":true,"prompt_max_tokens":32000}]}`)
	added, updated, err = r.RefreshFromModelList(body2)
	if err != nil {
		t.Fatalf("second RefreshFromModelList() error = %v", err)
	}
	if added != 0 || updated != 1 {
		t.Errorf("second refresh added=%d updated=%d, want 0/1", added, updated)
	}
	if m := r.Get("seed_m8"); m.MaxTokens != 32000 {
		t.Errorf("seed_m8 max tokens = %d, want 32000", m.MaxTokens)
	}
}

// TestRegistry_RefreshFromCapturedSample parses the captured live model_list
// payload when available in the repo.
func TestRegistry_RefreshFromCapturedSample(t *testing.T) {
	body, err := os.ReadFile("../../scripts/model_list_llm_raw_chat.json")
	if err != nil {
		t.Skip("captured sample not available")
	}
	r := NewRegistry()
	added, _, err := r.RefreshFromModelList(body)
	if err != nil {
		t.Fatalf("RefreshFromModelList(captured) error = %v", err)
	}
	t.Logf("captured sample merged, %d new remote models", added)
	// The captured list carries exactly 5 presets (seed_m8,
	// Doubao_1_5_thinking_pro, deepseek-R1, deepseek-V3, deepseek-V3-0324).
	if added != 5 {
		t.Errorf("added = %d, want 5 preset models from captured sample", added)
	}
	if len(r.List()) < len(requiredModels) {
		t.Error("registry shrank after refresh")
	}
}
