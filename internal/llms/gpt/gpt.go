package gpt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/martinbockt/esc-llm-webscraper/internal/llms"
	openai "github.com/sashabaranov/go-openai"
)

var _ = (llms.Plugin)(&gpt{})

type gpt struct {
	client       *openai.Client
	imageSupport bool
	model        string
	tools        []openai.Tool
	toolChoice   any
	guided       bool
	messages     []openai.ChatCompletionMessage
}

type functionChoice struct {
	Type     string `json:"type"`
	Function struct {
		Name string `json:"name"`
	} `json:"function"`
}

func New(client *openai.Client, model string, imageSupport bool) llms.Plugin {
	t := []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        llms.RoomsName,
				Description: llms.RoomsDescription,
				Parameters:  generateSchema(llms.RoomsResp{}),
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        llms.URLsName,
				Description: llms.URLsDescription,
				Parameters:  generateSchema(llms.UrlsResp{}),
			},
		},
	}

	return &gpt{
		client:       client,
		imageSupport: imageSupport,
		model:        model,
		tools:        t,
		toolChoice:   "required",
	}
}

func (g *gpt) AddPrompt(_ []byte, text, chatID, toolName string) {
	role := openai.ChatMessageRoleUser

	if g.guided {
		g.toolChoice = functionChoice{
			Type: "function",
			Function: struct {
				Name string `json:"name"`
			}{
				Name: llms.URLsName,
			},
		}
	}

	if len(g.messages) > 0 {
		role = openai.ChatMessageRoleTool
		g.toolChoice = "required"
	} else {
		g.messages = append(g.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: llms.SystemPrompt,
		}, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: "Only call one tool function at a time",
		})
	}

	g.messages = append(g.messages, openai.ChatCompletionMessage{
		Role:       role,
		Content:    text,
		ToolCallID: chatID,
		Name:       toolName,
	})
}

func (g *gpt) ExecutePrompt(ctx context.Context) ([]llms.LlmResposeWithChatID, time.Duration, int, error) {
	request := openai.ChatCompletionRequest{
		Model:      g.model,
		Messages:   g.messages,
		Tools:      g.tools,
		ToolChoice: g.toolChoice,
	}

	startTime := time.Now()
	resp, err := g.client.CreateChatCompletion(
		ctx,
		request,
	)
	duration := time.Since(startTime)
	if err != nil {
		return nil, duration, 0, fmt.Errorf("ChatCompletion error: %w", err)
	}

	totalTokens := resp.Usage.TotalTokens

	g.messages = append(g.messages, resp.Choices[0].Message)

	response := []llms.LlmResposeWithChatID{}
	if len(resp.Choices) > 0 {
		for _, toolCall := range resp.Choices[0].Message.ToolCalls {
			var resp interface{} = &llms.RoomsResp{}
			if toolCall.Function.Name == llms.URLsName {
				resp = &llms.UrlsResp{}
			}
			err = json.Unmarshal([]byte(toolCall.Function.Arguments), &resp)
			if err != nil {
				return nil, duration, totalTokens, fmt.Errorf("failed to unmarshal response: %w", err)
			}

			result := llms.LlmResposeWithChatID{
				ChatID:   toolCall.ID,
				ToolName: toolCall.Function.Name,
			}
			if urls, ok := resp.(*llms.UrlsResp); ok {
				result.URLs = urls.URLs
			} else if rooms, ok := resp.(*llms.RoomsResp); ok {
				result.Rooms = rooms.Rooms
			}

			response = append(response, result)
		}
	}

	return response, duration, totalTokens, nil
}

func (g *gpt) ResetChat() {
	g.toolChoice = "required"
	g.messages = []openai.ChatCompletionMessage{}
}

func (g *gpt) ModelName() string {
	return g.model
}

func (g *gpt) ImageSupport() bool {
	return g.imageSupport
}

func (g *gpt) RoomToolOnly() {
	g.guided = false

	g.toolChoice = functionChoice{
		Type: "function",
		Function: struct {
			Name string `json:"name"`
		}{
			Name: llms.RoomsName,
		},
	}
}

func (g *gpt) Guided(mode bool) {
	g.guided = mode
}
