package togetherai

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
	Messages         []Message          `json:"messages"`
	Model            string             `json:"model"`
	MaxTokens        int                `json:"max_tokens,omitempty"`
	Stop             []string           `json:"stop,omitempty"`
	Temperature      float64            `json:"temperature,omitempty"`
	TopP             float64            `json:"top_p,omitempty"`
	TokK             int32              `json:"tok_k,omitempty"`
	RepetionPenalty  int                `json:"repetition_penalty,omitempty"`
	Stream           bool               `json:"stream,omitempty"`
	Longprobs        uint8              `json:"longprobs,omitempty"` // 0 or 1
	Echo             bool               `json:"echo,omitempty"`
	N                int                `json:"n,omitempty"`
	MinP             float64            `json:"min_p,omitempty"`
	PresencePenalty  float64            `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64            `json:"frequency_penalty,omitempty"`
	LongBias         map[string]float64 `json:"long_bias,omitempty"`
	FunctionCall     *FunctionCall      `json:"function_call,omitempty"`
	ResponseFormat   *ResponseFormat    `json:"response_format,omitempty"`
	Tools            []Tool             `json:"tools,omitempty"`
	ToolChoice       string             `json:"tool_choice,omitempty"`
	SafetyModel      string             `json:"safety_model,omitempty"`
}

type Tool struct {
	Type     string   `json:"name,omitempty"`
	Function Function `json:"function,omitempty"`
}

type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ResponseFormat struct {
	Type   string                 `json:"type,omitempty"`
	Schema map[string]interface{} `json:"schema,omitempty"`
}

type FunctionCall struct {
	Name string `json:"name"`
}

type Role string

const (
	User      Role = "user"
	System    Role = "system"
	Assistant Role = "assistant"
)

type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

type Object string

const (
	Completion Object = "chat.completion"
)

type ChatCompletionResponse struct {
	ID        string     `json:"id"`
	Choices   []Choice   `json:"choices"`
	LogProbes *LogProbes `json:"logprobs"`
	Usage     *Usage     `json:"usage"`
	Created   int        `json:"created"`
	Model     string     `json:"model"`
	Object    Object     `json:"object"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type LogProbes struct {
	TokenIDs      []int    `json:"token_ids"`
	Tokens        []string `json:"tokens"`
	TokenLogProbs []int    `json:"token_logprobs"`
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
	Text         string          `json:"text"`
	Index        int             `json:"index"`
	Seed         uint64          `json:"seed"`
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
	Index    int              `json:"index"`
	Type     ToolCallType     `json:"type"`
	Function FunctionResponse `json:"function"`
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
