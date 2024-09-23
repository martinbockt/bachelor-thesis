package claude

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/martinbockt/esc-llm-webscraper/internal/llms"

	langchain "github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

var _ = (llms.Plugin)(&claude{})

type toolChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

type claude struct {
	imageSupport bool
	tools        []langchain.Tool
	model        *anthropic.LLM
	messages     []langchain.MessageContent
	toolChoice   toolChoice
	modelName    string
	guided       bool
}

func New(modelName string, token string, imageSupport bool) llms.Plugin {
	llm, err := anthropic.New(anthropic.WithModel(modelName), anthropic.WithToken(token))
	if err != nil {
		panic(fmt.Errorf("failed to create LLM: %w", err))
	}

	tools := []langchain.Tool{
		{
			Type: "function",
			Function: &langchain.FunctionDefinition{
				Name:        llms.RoomsName,
				Description: llms.RoomsDescription,
				Parameters:  generateSchemaMap(llms.RoomsResp{}),
			},
		},
		{
			Type: "function",
			Function: &langchain.FunctionDefinition{
				Name:        llms.URLsName,
				Description: llms.URLsDescription,
				Parameters:  generateSchemaMap(llms.UrlsResp{}),
			},
		},
	}

	return &claude{
		tools:        tools,
		imageSupport: imageSupport,
		model:        llm,
		modelName:    modelName,
		toolChoice: toolChoice{
			Type: "tool",
			Name: llms.URLsName,
		},
	}
}

func (c *claude) AddPrompt(image []byte, text, chatID, toolName string) {
	content := []langchain.ContentPart{
		langchain.TextPart(text),
	}

	if c.guided {
		c.toolChoice = toolChoice{
			Type: "function",
			Name: llms.URLsName,
		}
	}

	if chatID != "" {
		c.toolChoice = toolChoice{
			Type: "any",
		}

		content = []langchain.ContentPart{
			langchain.ToolCallResponse{
				ToolCallID: chatID,
				Name:       toolName,
				Content:    text,
			},
		}
	}

	if len(image) > 0 {
		content = append(content, langchain.BinaryPart("image/webp", image))
	}

	role := langchain.ChatMessageTypeHuman
	if len(c.messages) > 0 {
		role = langchain.ChatMessageTypeTool
	}

	c.messages = append(c.messages, langchain.MessageContent{
		Role:  role,
		Parts: content,
	})
}

func (c *claude) ExecutePrompt(ctx context.Context) ([]llms.LlmResposeWithChatID, time.Duration, int, error) {
	startTime := time.Now()
	resp, err := c.model.GenerateContent(ctx, c.messages, langchain.WithTools(c.tools), langchain.WithToolChoice(c.toolChoice), langchain.WithMaxTokens(8192))
	duration := time.Since(startTime)
	if err != nil {
		return nil, duration, 0, err
	}

	if len(resp.Choices) == 0 {
		return nil, duration, 0, errors.New("no choices returned")
	}

	llmResponseWithChatID := []llms.LlmResposeWithChatID{}

	for _, choice := range resp.Choices {
		for _, toolCall := range choice.ToolCalls {
			assistantResponse := langchain.MessageContent{
				Role: langchain.ChatMessageTypeAI,
				Parts: []langchain.ContentPart{
					langchain.ToolCall{
						ID:   toolCall.ID,
						Type: toolCall.Type,
						FunctionCall: &langchain.FunctionCall{
							Name:      toolCall.FunctionCall.Name,
							Arguments: toolCall.FunctionCall.Arguments,
						},
					},
				},
			}
			c.messages = append(c.messages, assistantResponse)

			var llmResponse interface{} = &llms.RoomsResp{}
			if toolCall.FunctionCall.Name == llms.URLsName {
				llmResponse = &llms.UrlsResp{}
			}

			err = json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &llmResponse)
			if err != nil {
				return nil, duration, 0, fmt.Errorf("failed to unmarshal response: %w", err)
			}

			result := llms.LlmResposeWithChatID{
				ToolName: toolCall.FunctionCall.Name,
				ChatID:   toolCall.ID,
			}

			if urls, ok := llmResponse.(*llms.UrlsResp); ok {
				result.URLs = urls.URLs
			} else if rooms, ok := llmResponse.(*llms.RoomsResp); ok {
				result.Rooms = rooms.Rooms
			}

			llmResponseWithChatID = append(llmResponseWithChatID, result)
		}
	}

	return llmResponseWithChatID, duration, 0, nil
}

func (c *claude) ModelName() string {
	return c.modelName
}

func (c *claude) ImageSupport() bool {
	return c.imageSupport
}

func (c *claude) ResetChat() {
	c.toolChoice = toolChoice{
		Type: "any",
	}
	c.messages = nil
}

func (c *claude) RoomToolOnly() {
	c.guided = false
	c.toolChoice = toolChoice{
		Type: "tool",
		Name: llms.RoomsName,
	}
}

func (c *claude) Guided(mode bool) {
	c.guided = mode
}
