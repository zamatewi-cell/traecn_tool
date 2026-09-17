package transformers

import (
	"testing"

	"github.com/zamatewi-cell/traecn_tool/internal/models"
	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
)

func TestRequestTransformer_NewRequestTransformer(t *testing.T) {
	rt := NewRequestTransformer()
	if rt == nil {
		t.Fatal("NewRequestTransformer() returned nil")
	}
}

func TestRequestTransformer_Transform_BasicRequest(t *testing.T) {
	rt := NewRequestTransformer()
	temp := float64(0.7)
	maxTokens := 100

	req := &OpenAIRequest{
		Model: "deepseek-chat",
		Messages: []OpenAIMessage{
			{Role: "system", Content: "You are a helpful assistant"},
			{Role: "user", Content: "Hello"},
		},
		Stream:      false,
		Temperature: &temp,
		MaxTokens:   &maxTokens,
	}

	traeReq, err := rt.Transform(req)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	if traeReq.ModelName != "DeepSeek-V4-Flash" {
		t.Errorf("Transform() ModelID = %v, want DeepSeek-V4-Flash", traeReq.ModelName)
	}
	if len(traeReq.Messages) != 2 {
		t.Errorf("Transform() Messages length = %v, want 2", len(traeReq.Messages))
	}
	if traeReq.Stream != false {
		t.Errorf("Transform() Stream = %v, want false", traeReq.Stream)
	}
	if traeReq.Messages[0].Role != "system" || traeReq.Messages[1].Role != "user" {
		t.Errorf("Transform() role mapping wrong: %+v", traeReq.Messages)
	}
}

func TestRequestTransformer_Transform_UnknownModel(t *testing.T) {
	rt := NewRequestTransformer()

	req := &OpenAIRequest{
		Model: "unknown-model-xyz",
		Messages: []OpenAIMessage{
			{Role: "user", Content: "Test"},
		},
		Stream: true,
	}

	traeReq, err := rt.Transform(req)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	// Unknown model falls back to the registry default
	if traeReq.ModelName != models.DefaultModel {
		t.Errorf("Transform() ModelID = %v, want %v", traeReq.ModelName, models.DefaultModel)
	}
}

func TestRequestTransformer_Transform_AllParameters(t *testing.T) {
	rt := NewRequestTransformer()
	temp := float64(0.8)
	maxTokens := 200
	topP := float64(0.9)
	freqPenalty := float64(0.5)
	presPenalty := float64(0.3)

	req := &OpenAIRequest{
		Model: "qwen-2.5-72b",
		Messages: []OpenAIMessage{
			{Role: "user", Content: "Test"},
		},
		Stream:           true,
		Temperature:      &temp,
		MaxTokens:        &maxTokens,
		TopP:             &topP,
		FrequencyPenalty: &freqPenalty,
		PresencePenalty:  &presPenalty,
		Stop:             []string{"END", "STOP"},
	}

	traeReq, err := rt.Transform(req)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	// Sampling parameters are accepted on input but not forwarded upstream
	// (the upstream chat payload has no sampling fields on the wire).
	if !traeReq.Stream {
		t.Error("Transform() Stream = false, want true")
	}
	if len(traeReq.Messages) != 1 || traeReq.Messages[0].Content != "Test" {
		t.Errorf("Transform() messages wrong: %+v", traeReq.Messages)
	}
}

func TestRequestTransformer_mapModel_KnownModels(t *testing.T) {
	rt := NewRequestTransformer()

	tests := []struct {
		openAIModel string
		wantTraeID  string
	}{
		{"deepseek-r1", "DeepSeek-V4-Pro"},
		{"deepseek-chat", "DeepSeek-V4-Flash"},
		{"glm", "GLM-5.3"},
		{"qwen", "Qwen3.7-Plus"},
		{"kimi", "Kimi-K3"},
		{"doubao", "Seed-Code"},
		{"DeepSeek-V4.1-Flash", "DeepSeek-V4.1-Flash"},
		{"DeepSeek-V4.1", "DeepSeek-V4.1-Flash"},
		{"totally-unknown-xyz", models.DefaultModel},
	}

	for _, tt := range tests {
		t.Run(tt.openAIModel, func(t *testing.T) {
			req := &OpenAIRequest{
				Model: tt.openAIModel,
				Messages: []OpenAIMessage{
					{Role: "user", Content: "Test"},
				},
			}
			traeReq, err := rt.Transform(req)
			if err != nil {
				t.Fatalf("Transform() error = %v", err)
			}
			if traeReq.ModelName != tt.wantTraeID {
				t.Errorf("mapModel(%v) = %v, want %v", tt.openAIModel, traeReq.ModelName, tt.wantTraeID)
			}
		})
	}
}

func TestResponseTransformer_NewResponseTransformer(t *testing.T) {
	rt := NewResponseTransformer()
	if rt == nil {
		t.Fatal("NewResponseTransformer() returned nil")
	}
}

func TestResponseTransformer_TransformChunk_BasicChunk(t *testing.T) {
	rt := NewResponseTransformer()

	traeChunk := &TraeResponse{
		ID: "12345",
		Choices: []TraeChoice{
			{
				Delta: &TraeDelta{
					Content: "Hello",
				},
			},
		},
	}

	model := "deepseek-v3"
	openAIResp := rt.TransformChunk(traeChunk, model)

	if openAIResp == nil {
		t.Fatal("TransformChunk() returned nil")
	}
	if openAIResp.Object != "chat.completion.chunk" {
		t.Errorf("TransformChunk() Object = %v, want chat.completion.chunk", openAIResp.Object)
	}
	if openAIResp.Model != model {
		t.Errorf("TransformChunk() Model = %v, want %v", openAIResp.Model, model)
	}
	if len(openAIResp.Choices) != 1 {
		t.Errorf("TransformChunk() Choices length = %v, want 1", len(openAIResp.Choices))
	}
	if openAIResp.Choices[0].Delta == nil {
		t.Error("TransformChunk() Delta is nil")
	}
}

func TestResponseTransformer_TransformChunk_EmptyChoices(t *testing.T) {
	rt := NewResponseTransformer()

	traeChunk := &TraeResponse{
		ID:      "12345",
		Choices: []TraeChoice{},
	}

	openAIResp := rt.TransformChunk(traeChunk, "test-model")
	if openAIResp != nil {
		t.Error("TransformChunk() should return nil for empty choices")
	}
}

func TestResponseTransformer_TransformComplete_BasicResponse(t *testing.T) {
	rt := NewResponseTransformer()

	finishReason := "stop"
	traeResp := &TraeResponse{
		ID: "12345",
		Choices: []TraeChoice{
			{
				Message: &TraeMessage{
					Role:    "assistant",
					Content: "Hello, I am an AI assistant",
				},
				FinishReason: &finishReason,
			},
		},
		Usage: &TraeUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	model := "deepseek-v3"
	openAIResp := rt.TransformComplete(traeResp, model)

	if openAIResp == nil {
		t.Fatal("TransformComplete() returned nil")
	}
	if openAIResp.Object != "chat.completion" {
		t.Errorf("TransformComplete() Object = %v, want chat.completion", openAIResp.Object)
	}
	if openAIResp.Model != model {
		t.Errorf("TransformComplete() Model = %v, want %v", openAIResp.Model, model)
	}
	if openAIResp.Choices[0].Message.Role != "assistant" {
		t.Errorf("TransformComplete() Message.Role = %v, want assistant", openAIResp.Choices[0].Message.Role)
	}
	if openAIResp.Usage == nil {
		t.Error("TransformComplete() Usage is nil")
	}
	if openAIResp.Usage.TotalTokens != 30 {
		t.Errorf("TransformComplete() Usage.TotalTokens = %v, want 30", openAIResp.Usage.TotalTokens)
	}
}

func TestResponseTransformer_TransformComplete_EmptyChoices(t *testing.T) {
	rt := NewResponseTransformer()

	traeResp := &TraeResponse{
		ID:      "12345",
		Choices: []TraeChoice{},
	}

	openAIResp := rt.TransformComplete(traeResp, "test-model")
	if openAIResp != nil {
		t.Error("TransformComplete() should return nil for empty choices")
	}
}

func TestOpenAIRequest_MarshalUnmarshal(t *testing.T) {
	temp := float64(0.7)
	req := &OpenAIRequest{
		Model: "deepseek-v3",
		Messages: []OpenAIMessage{
			{Role: "user", Content: "Hello", Name: "Alice"},
		},
		Stream:      true,
		Temperature: &temp,
	}

	// Test that all fields are accessible
	if req.Model != "deepseek-v3" {
		t.Errorf("Model = %v, want deepseek-v3", req.Model)
	}
	if len(req.Messages) != 1 {
		t.Errorf("Messages length = %v, want 1", len(req.Messages))
	}
	if req.Stream != true {
		t.Errorf("Stream = %v, want true", req.Stream)
	}
}

func TestTraeRequest_Fields(t *testing.T) {
	req := &TraeRequest{
		ModelID:        "ds_v3",
		ConversationID: "conv-123",
		Messages: []proxy.Message{
			{Role: "user", Content: "Test"},
		},
		Stream: true,
		Parameters: map[string]interface{}{
			"temperature": 0.7,
		},
	}

	if req.ModelID != "ds_v3" {
		t.Errorf("ModelID = %v, want ds_v3", req.ModelID)
	}
	if req.ConversationID != "conv-123" {
		t.Errorf("ConversationID = %v, want conv-123", req.ConversationID)
	}
	if !req.Stream {
		t.Error("Stream = false, want true")
	}
}
