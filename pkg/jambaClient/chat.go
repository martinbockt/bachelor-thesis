package jambaClient

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// https://docs.together.ai/reference/chat-completions-1

// ChatService provides methods to interact with the TogetherAI Chat API.
type ChatService struct {
	client *client
}

type ChatCompletionRequest struct {
	Model          string          `json:"model"`                     // Model name to use
	Messages       []Message       `json:"messages"`                  // List of messages (user, assistant, system)
	MaxTokens      int             `json:"max_tokens,omitempty"`      // Maximum number of tokens for the response
	Stop           []string        `json:"stop,omitempty"`            // Optional stop sequences
	Temperature    float64         `json:"temperature,omitempty"`     // Sampling temperature
	TopP           float64         `json:"top_p,omitempty"`           // Top-p sampling threshold
	N              int             `json:"n,omitempty"`               // Number of responses to generate
	Stream         bool            `json:"stream,omitempty"`          // Whether to stream responses
	Tools          []Tool          `json:"tools,omitempty"`           // List of tools for the model to use
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"` // Response format (text or JSON)
	Documents      []Document      `json:"documents,omitempty"`       // External documents for context
}

type Document struct {
	Content  string     `json:"content"`            // Main body of the document
	Metadata []Metadata `json:"metadata,omitempty"` // Optional metadata for the document
}

type Metadata struct {
	Key   string `json:"key"`   // Type of metadata (e.g., "author", "date")
	Value string `json:"value"` // Value of the metadata
}

type Tool struct {
	Type     string   `json:"type,omitempty"`
	Function Function `json:"function,omitempty"`
}

type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ResponseFormat struct {
	Type   string                 `json:"type,omitempty"`
	Schema map[string]interface{} `json:"description,omitempty"`
}

type Role string

const (
	RoleUser      Role = "user"
	RoleSystem    Role = "system"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // required for tool calls
}

type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type FinishReason string

const (
	StopReason         FinishReason = "stop"
	EosReason          FinishReason = "eos"
	LengthReason       FinishReason = "length"
	ToolCallsReason    FinishReason = "tool_calls"
	FunctionCallReason FinishReason = "function_call"
)

type Choice struct {
	Index        int             `json:"index"`
	FinishReason FinishReason    `json:"finish_reason"`
	Message      ResponseMessage `json:"message"`
}

type ResponseMessage struct {
	Role      string     `json:"role"`
	Content   *string    `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls"`
}

type ToolCallType string

const (
	ToolCallTypeFunction ToolCallType = "function"
)

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type,omitempty"`
	Function FunctionResponse `json:"function,omitempty"`
}

type FunctionResponse struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// NewChatService initializes a new ChatService.
func NewChatService(logger *zap.Logger, apiKey string) *ChatService {
	return &ChatService{client: newClient(logger, apiKey)}
}

// CreateChatCompletion creates a chat completion using the TogetherAI API.
func (s *ChatService) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error) {
	res := ChatCompletionResponse{}
	err := s.client.post(ctx, "/chat/completions", req, &res)
	if err != nil {
		return res, fmt.Errorf("failed to create chat completion: %w", err)
	}

	return res, nil
}
