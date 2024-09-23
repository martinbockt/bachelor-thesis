package mistral

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/martinbockt/esc-llm-webscraper/internal/llms"

	langchain "github.com/tmc/langchaingo/llms"
	mistralSDK "github.com/tmc/langchaingo/llms/mistral"
)

var _ = (llms.Plugin)(&mistral{})

type mistral struct {
	imageSupport bool
	tools        []langchain.Tool
	model        *mistralSDK.Model
	messages     []langchain.MessageContent
	modelName    string
	guided       bool
}

func New(model string, token string, imageSupport bool) llms.Plugin {
	llm, err := mistralSDK.New(mistralSDK.WithModel(model), mistralSDK.WithAPIKey(token))
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

	return &mistral{
		tools:        tools,
		imageSupport: imageSupport,
		model:        llm,
		modelName:    model,
	}
}

func (m *mistral) AddPrompt(image []byte, text, chatID, toolName string) {
	content := []langchain.ContentPart{
		langchain.TextPart(llms.SystemPrompt),
		langchain.TextPart(text),
	}

	if m.guided {
		m.tools = []langchain.Tool{
			{
				Type: "function",
				Function: &langchain.FunctionDefinition{
					Name:        llms.URLsName,
					Description: llms.URLsDescription,
					Parameters:  generateSchemaMap(llms.UrlsResp{}),
				},
			},
		}
	}

	if chatID != "" {
		m.tools = []langchain.Tool{
			{
				Type: "function",
				Function: &langchain.FunctionDefinition{
					Name:        llms.URLsName,
					Description: llms.URLsDescription,
					Parameters:  generateSchemaMap(llms.UrlsResp{}),
				},
			},
			{
				Type: "function",
				Function: &langchain.FunctionDefinition{
					Name:        llms.RoomsName,
					Description: llms.RoomsDescription,
					Parameters:  generateSchemaMap(llms.RoomsResp{}),
				},
			},
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
	if len(m.messages) > 0 {
		role = langchain.ChatMessageTypeTool
	}

	m.messages = append(m.messages, langchain.MessageContent{
		Role:  role,
		Parts: content,
	})
}

func (m *mistral) ExecutePrompt(ctx context.Context) ([]llms.LlmResposeWithChatID, time.Duration, int, error) {
	startTime := time.Now()
	resp, err := m.model.GenerateContent(ctx, m.messages, langchain.WithTools(m.tools), langchain.WithToolChoice("any"), langchain.WithMaxTokens(40000))
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
			m.messages = append(m.messages, assistantResponse)
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

func (m *mistral) ModelName() string {
	return m.modelName
}

func (m *mistral) ImageSupport() bool {
	return m.imageSupport
}

func (m *mistral) ResetChat() {
	m.tools = []langchain.Tool{
		{
			Type: "function",
			Function: &langchain.FunctionDefinition{
				Name:        llms.URLsName,
				Description: llms.URLsDescription,
				Parameters:  generateSchemaMap(llms.UrlsResp{}),
			},
		},
		{
			Type: "function",
			Function: &langchain.FunctionDefinition{
				Name:        llms.RoomsName,
				Description: llms.RoomsDescription,
				Parameters:  generateSchemaMap(llms.RoomsResp{}),
			},
		},
	}

	m.messages = nil
}

func (m *mistral) Guided(mode bool) {
	m.guided = mode
}

func (m *mistral) RoomToolOnly() {
	m.guided = false
	m.tools = []langchain.Tool{
		{
			Type: "function",
			Function: &langchain.FunctionDefinition{
				Name:        llms.RoomsName,
				Description: llms.RoomsDescription,
				Parameters:  generateSchemaMap(llms.RoomsResp{}),
			},
		},
	}
}
