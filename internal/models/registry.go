package models

import (
	"sort"
	"strings"
	"sync"
)

// ChannelType specifies which upstream transport channel serves this model.
type ChannelType int

const (
	// ChannelLegacyHTTPS routes to /api/ide/v1/llm_raw_chat (the 5 preset models).
	ChannelLegacyHTTPS ChannelType = iota
	// ChannelAgentTask routes to /api/agent/v3/create_agent_task (16 new builtin models).
	ChannelAgentTask
)

// ModelInfo describes one registered model.
type ModelInfo struct {
	ModelID       string      `json:"model_id"`
	DisplayName   string      `json:"display_name"`
	Provider      string      `json:"provider"`
	MaxTokens     int         `json:"max_tokens"`
	ContextWindow int         `json:"context_window"`
	Aliases       []string    `json:"aliases,omitempty"`
	Source        string      `json:"source"` // "builtin" | "remote"
	// UpstreamID is the model_name the Trae CN backend actually accepts
	// on llm_raw_chat (empty = resolved dynamically after a registry refresh).
	UpstreamID string      `json:"upstream_id,omitempty"`
	Channel    ChannelType `json:"channel"`
}

// DefaultModel is used when a request omits the model or names an unknown one.
const DefaultModel = "Seed-Code"

// DefaultUpstreamModel is the fallback upstream model_name when no mapping
// exists (the live preset default, verified against llm_raw_chat).
const DefaultUpstreamModel = "seed_m8"

// builtinModels is the hardcoded registry of 16 new generation IDE models.
// MaxTokens / ContextWindow are conservative metadata defaults; the dynamic
// refresher overwrites them with real upstream values.
var builtinModels = []ModelInfo{
	// ByteDance / Doubao Seed family
	{ModelID: "Seed-Code", DisplayName: "Seed Code", Provider: "ByteDance", MaxTokens: 16000, ContextWindow: 256000, Aliases: []string{"doubao-seed-code", "doubao-seed", "doubao-code", "doubao", "seed"}, Source: "builtin", Channel: ChannelAgentTask},
	{ModelID: "Seed-Evolving", DisplayName: "Seed Evolving", Provider: "ByteDance", MaxTokens: 16000, ContextWindow: 256000, Source: "builtin", Channel: ChannelAgentTask},
	{ModelID: "Seed-2.1-Pro-0915", DisplayName: "Seed 2.1 Pro (0915)", Provider: "ByteDance", MaxTokens: 16000, ContextWindow: 256000, Source: "builtin", Channel: ChannelAgentTask},
	{ModelID: "Seed-2.1-Turbo", DisplayName: "Seed 2.1 Turbo", Provider: "ByteDance", MaxTokens: 16000, ContextWindow: 256000, Source: "builtin", Channel: ChannelAgentTask},

	// Zhipu GLM family
	{ModelID: "GLM-5.3-Flash", DisplayName: "GLM 5.3 Flash", Provider: "Zhipu", MaxTokens: 8192, ContextWindow: 128000, Source: "builtin", Channel: ChannelAgentTask},
	{ModelID: "GLM-5.3", DisplayName: "GLM 5.3", Provider: "Zhipu", MaxTokens: 16384, ContextWindow: 128000, Aliases: []string{"glm-5", "glm"}, Source: "builtin", Channel: ChannelAgentTask},
	{ModelID: "GLM-5.2", DisplayName: "GLM 5.2", Provider: "Zhipu", MaxTokens: 16384, ContextWindow: 128000, Aliases: []string{"glm-5.2"}, Source: "builtin", Channel: ChannelAgentTask},

	// DeepSeek family
	{ModelID: "DeepSeek-V4.1-Flash", DisplayName: "DeepSeek V4.1 Flash", Provider: "DeepSeek", MaxTokens: 16384, ContextWindow: 128000, Aliases: []string{"DeepSeek-V4.1", "deepseek-v4.1-flash"}, Source: "builtin", Channel: ChannelAgentTask},
	{ModelID: "DeepSeek-V4-Flash", DisplayName: "DeepSeek V4 Flash", Provider: "DeepSeek", MaxTokens: 8192, ContextWindow: 128000, Aliases: []string{"deepseek-v4", "deepseek-chat"}, Source: "builtin", Channel: ChannelAgentTask},
	{ModelID: "DeepSeek-V4-Pro", DisplayName: "DeepSeek V4 Pro", Provider: "DeepSeek", MaxTokens: 16384, ContextWindow: 128000, Aliases: []string{"deepseek-r1", "deepseek-reasoner"}, Source: "builtin", Channel: ChannelAgentTask},

	// Moonshot Kimi family
	{ModelID: "Kimi-K3", DisplayName: "Kimi K3", Provider: "Moonshot", MaxTokens: 16384, ContextWindow: 262144, Aliases: []string{"kimi", "kimi-k3"}, Source: "builtin", Channel: ChannelAgentTask},
	{ModelID: "Kimi-K2.8-Preview", DisplayName: "Kimi K2.8 Preview", Provider: "Moonshot", MaxTokens: 16384, ContextWindow: 262144, Source: "builtin", Channel: ChannelAgentTask},

	// MiniMax family
	{ModelID: "MiniMax-M3", DisplayName: "MiniMax M3", Provider: "MiniMax", MaxTokens: 16384, ContextWindow: 1000000, Aliases: []string{"minimax", "minimax-m3"}, Source: "builtin", Channel: ChannelAgentTask},

	// Alibaba Qwen family
	{ModelID: "Qwen3.8-Flash", DisplayName: "Qwen 3.8 Flash", Provider: "Alibaba", MaxTokens: 8192, ContextWindow: 131072, Source: "builtin", Channel: ChannelAgentTask},
	{ModelID: "Qwen3.8-Max", DisplayName: "Qwen 3.8 Max", Provider: "Alibaba", MaxTokens: 16384, ContextWindow: 262144, Source: "builtin", Channel: ChannelAgentTask},
	{ModelID: "Qwen3.7-Plus", DisplayName: "Qwen 3.7 Plus", Provider: "Alibaba", MaxTokens: 16384, ContextWindow: 131072, Aliases: []string{"qwen"}, Source: "builtin", Channel: ChannelAgentTask},
}

// PresetModels holds the 5 live preset models verified against llm_raw_chat.
var PresetModels = []ModelInfo{
	{ModelID: "seed_m8", DisplayName: "Doubao-1.5-pro", Provider: "ByteDance", MaxTokens: 16000, ContextWindow: 256000, Aliases: []string{"doubao-1.5-pro", "doubao_1_5_pro"}, Source: "builtin", UpstreamID: "seed_m8", Channel: ChannelLegacyHTTPS},
	{ModelID: "Doubao_1_5_thinking_pro", DisplayName: "Doubao-1.5-thinking-pro", Provider: "ByteDance", MaxTokens: 16000, ContextWindow: 256000, Aliases: []string{"doubao-1.5-thinking-pro", "doubao_1_5_thinking_pro"}, Source: "builtin", UpstreamID: "Doubao_1_5_thinking_pro", Channel: ChannelLegacyHTTPS},
	{ModelID: "deepseek-R1", DisplayName: "DeepSeek-Reasoner (R1)", Provider: "DeepSeek", MaxTokens: 16384, ContextWindow: 128000, Aliases: []string{"deepseek-r1"}, Source: "builtin", UpstreamID: "deepseek-R1", Channel: ChannelLegacyHTTPS},
	{ModelID: "deepseek-V3", DisplayName: "DeepSeek-V3", Provider: "DeepSeek", MaxTokens: 16384, ContextWindow: 128000, Aliases: []string{"deepseek-v3"}, Source: "builtin", UpstreamID: "deepseek-V3", Channel: ChannelLegacyHTTPS},
	{ModelID: "deepseek-V3-0324", DisplayName: "DeepSeek-V3-0324", Provider: "DeepSeek", MaxTokens: 16384, ContextWindow: 128000, Aliases: []string{"deepseek-v3-0324"}, Source: "builtin", UpstreamID: "deepseek-V3-0324", Channel: ChannelLegacyHTTPS},
}

// Normalize canonicalizes client-supplied model names for matching:
// lowercase, trimmed, underscores converted to dashes.
func Normalize(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.ReplaceAll(n, "_", "-")
	n = strings.Join(strings.Fields(n), "-")
	return n
}

// Registry is the thread-safe model registry: builtin hardcoded entries
// merged with dynamically refreshed upstream entries.
type Registry struct {
	mu      sync.RWMutex
	order   []string // canonical ModelIDs in stable order
	models  map[string]*ModelInfo
	aliases map[string]string // normalized alias -> ModelID
}

// NewRegistry creates a registry preloaded with the builtin model list.
func NewRegistry() *Registry {
	r := &Registry{
		models:  make(map[string]*ModelInfo),
		aliases: make(map[string]string),
	}
	for _, m := range builtinModels {
		r.Upsert(m)
	}
	return r
}

// Upsert inserts or updates a model and its aliases.
func (r *Registry) Upsert(m ModelInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if m.DisplayName == "" {
		m.DisplayName = m.ModelID
	}
	if existing, ok := r.models[m.ModelID]; ok {
		// Preserve upstream-refreshed metadata when re-upserting builtins.
		if m.MaxTokens == 0 {
			m.MaxTokens = existing.MaxTokens
		}
		if m.ContextWindow == 0 {
			m.ContextWindow = existing.ContextWindow
		}
		*existing = m
	} else {
		r.order = append(r.order, m.ModelID)
		mc := m
		r.models[m.ModelID] = &mc
	}
	for _, a := range m.Aliases {
		r.aliases[Normalize(a)] = m.ModelID
	}
}

// RegisterAlias maps an additional alias onto an existing model.
func (r *Registry) RegisterAlias(alias, modelID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.models[modelID]; ok {
		r.aliases[Normalize(alias)] = modelID
	}
}

// Resolve finds a model by exact ID, case-insensitive ID, alias, or unique
// prefix. Returns nil when nothing matches.
func (r *Registry) Resolve(name string) *ModelInfo {
	if name == "" {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. Exact ModelID.
	if m, ok := r.models[name]; ok {
		return m
	}

	norm := Normalize(name)

	// 2. Case-insensitive / normalized ModelID.
	for id, m := range r.models {
		if Normalize(id) == norm {
			return m
		}
	}

	// 3. Alias table.
	if id, ok := r.aliases[norm]; ok {
		if m, ok := r.models[id]; ok {
			return m
		}
	}

	// 4. Unique normalized-prefix match ("glm-5.3-f" -> GLM-5.3-Flash).
	var match *ModelInfo
	count := 0
	for id, m := range r.models {
		if strings.HasPrefix(Normalize(id), norm) {
			match = m
			count++
		}
	}
	if count == 1 {
		return match
	}
	return nil
}

// UpstreamFor resolves a client-supplied model name to the upstream
// config_name the Trae CN backend accepts. Fallback chain: resolved
// model's UpstreamID -> the name itself when it is a known live (remote)
// model -> DefaultUpstreamModel.
func (r *Registry) UpstreamFor(name string) string {
	if m := r.Resolve(name); m != nil {
		if m.UpstreamID != "" {
			return m.UpstreamID
		}
		if m.Source == "remote" {
			return m.ModelID
		}
	}
	return DefaultUpstreamModel
}

// Get returns a model by canonical ID.
func (r *Registry) Get(modelID string) *ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.models[modelID]
}

// List returns all registered models in stable (provider, id) order.
func (r *Registry) List() []ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]ModelInfo, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, *r.models[id])
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Provider != out[j].Provider {
			return out[i].Provider < out[j].Provider
		}
		return out[i].ModelID < out[j].ModelID
	})
	return out
}

// Aliases returns a copy of the alias table (normalized alias -> ModelID).
func (r *Registry) Aliases() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]string, len(r.aliases))
	for k, v := range r.aliases {
		out[k] = v
	}
	return out
}

// NewDefaultRegistry creates a registry preloaded with both 16 builtin models
// and 5 legacy preset models (21 models total), ensuring /v1/models and model
// resolution are complete out of the box.
func NewDefaultRegistry() *Registry {
	r := NewRegistry()
	for _, m := range PresetModels {
		r.Upsert(m)
	}
	return r
}

var defaultRegistry = NewDefaultRegistry()

// Default returns the process-wide shared registry.
func Default() *Registry {
	return defaultRegistry
}

// FindModel resolves a client-supplied name against the default registry.
func FindModel(name string) *ModelInfo {
	return defaultRegistry.Resolve(name)
}

// ModelNames returns all canonical model IDs of the default registry.
func ModelNames() []string {
	list := defaultRegistry.List()
	names := make([]string, len(list))
	for i, m := range list {
		names[i] = m.ModelID
	}
	return names
}

// PresetModelNames holds the 5 live preset models verified against llm_raw_chat.
var PresetModelNames = []string{
	"seed_m8",
	"Doubao_1_5_thinking_pro",
	"deepseek-R1",
	"deepseek-V3",
	"deepseek-V3-0324",
}

// ResolveChannel determines which channel (legacy HTTPS vs AgentTask) a request should route to.
func ResolveChannel(modelName string) ChannelType {
	return defaultRegistry.ResolveChannel(modelName)
}

// ResolveChannel resolves the model's assigned communication channel.
func (r *Registry) ResolveChannel(modelName string) ChannelType {
	norm := Normalize(modelName)
	if norm == "" {
		return ChannelLegacyHTTPS
	}

	// 1. 5 Legacy preset models route to legacy HTTPS llm_raw_chat.
	switch norm {
	case "seed-m8", "doubao-1-5-pro", "doubao_1_5_pro",
		"doubao-1-5-thinking-pro", "doubao_1_5_thinking_pro",
		"deepseek-r1", "deepseek-v3", "deepseek-v3-0324":
		return ChannelLegacyHTTPS
	}

	// 2. 16 new generation builtin models (and ecosystem aliases) strictly route to AgentTask channel.
	switch norm {
	case "seed-code", "doubao-seed-code", "doubao-seed", "doubao-code", "doubao", "seed",
		"seed-evolving", "seed-2-1-pro-0915", "seed-2-1-turbo",
		"deepseek-v4-1-flash", "deepseek-v4.1-flash", "deepseek-v4.1",
		"deepseek-v4-flash", "deepseek-v4", "deepseek-chat",
		"deepseek-v4-pro", "deepseek-reasoner",
		"glm-5-3", "glm-5.3", "glm-5", "glm",
		"glm-5-3-flash", "glm-5.3-flash",
		"glm-5-2", "glm-5.2",
		"glm-5-1", "glm-5.1",
		"glm-4-7", "glm-4.7",
		"kimi-k3", "kimi", "kimi-k2", "kimi-k2-8-preview", "kimi-k2.8-preview",
		"minimax-m3", "minimax", "minimax-m2-5", "minimax-m2.5", "minimax-m2-1", "minimax-m2.1",
		"qwen-3-7-plus", "qwen3.7-plus", "qwen", "qwen-3-8-flash", "qwen3.8-flash", "qwen-3-8-max", "qwen3.8-max",
		"qwen-3-5", "qwen3.5-plus", "qwen3-coder", "qwen3-coder-next",
		"gemini-3-1-pro", "gemini-3.1-pro", "gpt-5-mini":
		return ChannelAgentTask
	}

	// 3. Query registry if model exists.
	if m := r.Resolve(modelName); m != nil {
		if m.Channel == ChannelLegacyHTTPS {
			return ChannelLegacyHTTPS
		}
		if m.Channel == ChannelAgentTask {
			return ChannelAgentTask
		}
		if m.Source == "remote" {
			for _, p := range PresetModelNames {
				if Normalize(p) == norm {
					return ChannelLegacyHTTPS
				}
			}
		}
	}

	// 4. Fall back to AgentTask for unknown modern models.
	return ChannelAgentTask
}

