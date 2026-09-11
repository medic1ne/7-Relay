package router

import (
	"encoding/json"
	"fmt"
)

// FormatTranslator handles request/response translation between providers
type FormatTranslator struct{}

func NewFormatTranslator() *FormatTranslator {
	return &FormatTranslator{}
}

// ProviderRequest is the internal normalized request format
type ProviderRequest struct {
	Model    string          `json:"model"`
	Messages []ProviderMsg   `json:"messages"`
	Stream   bool            `json:"stream,omitempty"`
	Tools    []Tool          `json:"tools,omitempty"`
	MaxTokens int            `json:"max_tokens,omitempty"`
	Temperature *float64     `json:"temperature,omitempty"`
}

type ProviderMsg struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content,omitempty"`
	Name    string      `json:"name,omitempty"`
	// Claude-specific
	Thinking *ThinkingConfig `json:"thinking,omitempty"`
	// Tool call fields
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID   string     `json:"tool_call_id,omitempty"`
}

type ThinkingConfig struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// --- OpenAI Format ---

type OpenAIRequest struct {
	Model     string          `json:"model"`
	Messages  []OpenAIMessage `json:"messages"`
	Stream    bool            `json:"stream,omitempty"`
	Tools     []OpenAITool    `json:"tools,omitempty"`
	MaxTokens *int           `json:"max_tokens,omitempty"`
	Temp      *float64       `json:"temperature,omitempty"`
}

type OpenAIMessage struct {
	Role       string        `json:"role"`
	Content    interface{}   `json:"content,omitempty"`
	Name       string        `json:"name,omitempty"`
	ToolCalls  []ToolCall    `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}

type OpenAITool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type OpenAIResponse struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage   `json:"usage"`
}

type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type OpenAIStreamChunk struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []OpenAIStreamDelta `json:"choices"`
}

type OpenAIStreamDelta struct {
	Index        int              `json:"index"`
	Delta        StreamDelta      `json:"delta"`
	FinishReason *string          `json:"finish_reason"`
}

type StreamDelta struct {
	Role    string     `json:"role,omitempty"`
	Content string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// --- Claude (Anthropic) Format ---

type ClaudeRequest struct {
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
	Messages   []ClaudeMessage `json:"messages"`
	System    interface{}    `json:"system,omitempty"`
	Stream    bool           `json:"stream,omitempty"`
	Tools     []ClaudeTool   `json:"tools,omitempty"`
	Thinking  *ClaudeThinking `json:"thinking,omitempty"`
}

type ClaudeMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type ClaudeTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type ClaudeThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

type ClaudeResponse struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Role       string          `json:"role"`
	Content    []ClaudeContent `json:"content"`
	Model      string          `json:"model"`
	StopReason string          `json:"stop_reason"`
	Usage      ClaudeUsage     `json:"usage"`
}

type ClaudeContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// For tool use
	ID    string      `json:"id,omitempty"`
	Name  string      `json:"name,omitempty"`
	Input interface{} `json:"input,omitempty"`
}

type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// --- Gemini Format ---

type GeminiRequest struct {
	Contents         []GeminiContent  `json:"contents"`
	GenerationConfig *GeminiGenConfig `json:"generationConfig,omitempty"`
	Tools            []GeminiTool     `json:"tools,omitempty"`
}

type GeminiContent struct {
	Role  string         `json:"role"`
	Parts []GeminiPart   `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text,omitempty"`
	// For tool calls
	FunctionCall *GeminiFunctionCall `json:"functionCall,omitempty"`
}

type GeminiFunctionCall struct {
	Name   string      `json:"name"`
	Args   interface{} `json:"args"`
}

type GeminiGenConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
}

type GeminiTool struct {
	FunctionDeclarations []GeminiFuncDecl `json:"functionDeclarations"`
}

type GeminiFuncDecl struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type GeminiResponse struct {
	Candidates    []GeminiCandidate `json:"candidates"`
	UsageMetadata GeminiUsage       `json:"usageMetadata"`
}

type GeminiCandidate struct {
	Content      GeminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

type GeminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// ==================== TRANSLATION ====================

// ToClaudeRequest translates from internal format to Claude API format
func (ft *FormatTranslator) ToClaudeRequest(req ProviderRequest) (*ClaudeRequest, error) {
	// Extract system message
	var systemMsg string
	var claudeMessages []ClaudeMessage

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			// Claude uses separate system field
			if s, ok := msg.Content.(string); ok {
				systemMsg = s
			}
			continue
		}
		claudeMsg := ClaudeMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
		claudeMessages = append(claudeMessages, claudeMsg)
	}

	maxTokens := 8192
	if req.MaxTokens > 0 {
		maxTokens = req.MaxTokens
	}

	result := &ClaudeRequest{
		Model:     req.Model,
		MaxTokens: maxTokens,
		Messages:   claudeMessages,
		Stream:    req.Stream,
	}

	if systemMsg != "" {
		result.System = systemMsg
	}

	// Convert tools
	if len(req.Tools) > 0 {
		var claudeTools []ClaudeTool
		for _, t := range req.Tools {
			claudeTools = append(claudeTools, ClaudeTool{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				InputSchema: t.Function.Parameters,
			})
		}
		result.Tools = claudeTools
	}

	return result, nil
}

// ToGeminiRequest translates from internal format to Gemini API format
func (ft *FormatTranslator) ToGeminiRequest(req ProviderRequest) (*GeminiRequest, error) {
	var contents []GeminiContent

	for _, msg := range req.Messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		if msg.Role == "system" {
			// Gemini uses systemInstruction (part of GenerationConfig in v1beta)
			// For now, inject as first user message with prefix
			text := ""
			if s, ok := msg.Content.(string); ok {
				text = fmt.Sprintf("[System Instruction] %s\n\n", s)
			}
			contents = append(contents, GeminiContent{
				Role:  "user",
				Parts: []GeminiPart{{Text: text}},
			})
			continue
		}

		text := ""
		switch v := msg.Content.(type) {
		case string:
			text = v
		case nil:
			text = ""
		default:
			b, _ := json.Marshal(v)
			text = string(b)
		}

		contents = append(contents, GeminiContent{
			Role:  role,
			Parts: []GeminiPart{{Text: text}},
		})
	}

	result := &GeminiRequest{
		Contents: contents,
	}

	if req.Temperature != nil {
		result.GenerationConfig = &GeminiGenConfig{
			Temperature: req.Temperature,
		}
	}

	// Convert tools
	if len(req.Tools) > 0 {
		var decls []GeminiFuncDecl
		for _, t := range req.Tools {
			decls = append(decls, GeminiFuncDecl{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			})
		}
		result.Tools = []GeminiTool{{FunctionDeclarations: decls}}
	}

	return result, nil
}

// FromClaudeResponse translates Claude response to OpenAI format
func (ft *FormatTranslator) FromClaudeResponse(claudeResp *ClaudeResponse) *ChatResponse {
	var content string
	var toolCalls []ToolCall

	for _, c := range claudeResp.Content {
		switch c.Type {
		case "text":
			content += c.Text
		case "tool_use":
			args, _ := json.Marshal(c.Input)
			toolCalls = append(toolCalls, ToolCall{
				ID:   c.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      c.Name,
					Arguments: string(args),
				},
			})
		}
	}

	msg := Message{
		Role:    "assistant",
		Content: content,
	}

	return &ChatResponse{
		ID:      claudeResp.ID,
		Object:  "chat.completion",
		Model:   claudeResp.Model,
		Choices: []Choice{
			{
				Index:        0,
				Message:      msg,
				FinishReason: mapStopReason(claudeResp.StopReason),
			},
		},
		Usage: Usage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
	}
}

// FromGeminiResponse translates Gemini response to OpenAI format
func (ft *FormatTranslator) FromGeminiResponse(geminiResp *GeminiResponse) *ChatResponse {
	if len(geminiResp.Candidates) == 0 {
		return &ChatResponse{
			Choices: []Choice{{FinishReason: "stop"}},
		}
	}

	candidate := geminiResp.Candidates[0]
	var content string

	for _, part := range candidate.Content.Parts {
		content += part.Text
	}

	return &ChatResponse{
		Object:  "chat.completion",
		Choices: []Choice{
			{
				Index:        0,
				Message:      Message{Role: "assistant", Content: content},
				FinishReason: mapGeminiStopReason(candidate.FinishReason),
			},
		},
		Usage: Usage{
			PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
		},
	}
}

func mapStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "stop_sequence":
		return "stop"
	case "tool_use":
		return "tool_calls"
	case "max_tokens":
		return "length"
	default:
		return reason
	}
}

func mapGeminiStopReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	default:
		return "stop"
	}
}
