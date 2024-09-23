package jamba

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/martinbockt/esc-llm-webscraper/internal/llms"
	"github.com/martinbockt/esc-llm-webscraper/pkg/jambaClient"
)

var _ = (llms.Plugin)(&jamba{})

type jamba struct {
	client       *jambaClient.ChatService
	req          jambaClient.ChatCompletionRequest
	imageSupport bool
	guided       bool
}

func New(jambaClientClient *jambaClient.ChatService, modelName string, _ float32, imageSupport bool) llms.Plugin {
	req := jambaClient.ChatCompletionRequest{
		Model: modelName,
		Tools: []jambaClient.Tool{
			{
				Type: "function",
				Function: jambaClient.Function{
					Name:        llms.RoomsName,
					Description: llms.RoomsDescription,
					Parameters:  generateSchemaMap(llms.RoomsResp{}),
				},
			},
			{
				Type: "function",
				Function: jambaClient.Function{
					Name:        llms.URLsName,
					Description: llms.URLsDescription,
					Parameters:  generateSchemaMap(llms.UrlsResp{}),
				},
			},
		},
		ResponseFormat: &jambaClient.ResponseFormat{
			Type: "json_object",
		},
	}

	return &jamba{
		client:       jambaClientClient,
		req:          req,
		imageSupport: imageSupport,
	}
}

func (j *jamba) AddPrompt(_ []byte, text, toolCallID, _ string) {
	if toolCallID == "" {
		messages := []jambaClient.Message{
			{
				Role:    jambaClient.RoleSystem,
				Content: llms.SystemPrompt,
			},
			{
				Role:    jambaClient.RoleSystem,
				Content: "If not doing a toolcall, respond in following format: {\"rooms\":[{\"name\":\"The Secret Lab\",\"description\":\"Enter the mysterious lab of a mad scientist. Can you uncover the secrets and escape before time runs out?\",\"players_min\":2,\"players_max\":6,\"duration\":60,\"booking_url\":\"https://example.com/book/secret-lab\",\"detail_page_url\":\"https://example.com/rooms/secret-lab\",\"image_url\":\"https://example.com/images/secret-lab.jpg\",\"genre\":\"Science Fiction\",\"difficulty\":\"Medium\"},{\"name\":\"Pharaoh's Tomb\",\"description\":\"Trapped inside the tomb of an ancient Pharaoh, you must solve the riddles and find the way out before you are sealed inside forever.\",\"players_min\":4,\"players_max\":8,\"duration\":90,\"booking_url\":\"https://example.com/book/pharaohs-tomb\",\"detail_page_url\":\"https://example.com/rooms/pharaohs-tomb\",\"image_url\":\"https://example.com/images/pharaohs-tomb.jpg\",\"genre\":\"Egypt\",\"difficulty\":\"Hard\"},{\"name\":\"Haunted Mansion\",\"description\":\"A ghostly adventure awaits inside this eerie mansion. Can you solve the mystery of the haunted estate and escape its grasp?\",\"players_min\":3,\"players_max\":5,\"duration\":75,\"booking_url\":\"https://example.com/book/haunted-mansion\",\"detail_page_url\":\"https://example.com/rooms/haunted-mansion\",\"image_url\":\"https://example.com/images/haunted-mansion.jpg\",\"genre\":\"Horror\",\"difficulty\":\"Easy\"}]}",
			},
			{
				Role:    jambaClient.RoleSystem,
				Content: "Do not request a url you already have access to.",
			},
		}
		if j.guided {
			messages = append(messages, jambaClient.Message{
				Role:    jambaClient.RoleSystem,
				Content: fmt.Sprintf("Use the %s tool/function call in your first response", llms.URLsName),
			})
		}
		j.req.Messages = append(j.req.Messages, messages...)
	}

	message := jambaClient.Message{
		Role:    jambaClient.RoleUser,
		Content: text,
	}

	if toolCallID != "" {
		message.Role = jambaClient.RoleTool
		message.ToolCallID = toolCallID
	}

	j.req.Messages = append(j.req.Messages, message)
}

func (j *jamba) ExecutePrompt(ctx context.Context) ([]llms.LlmResposeWithChatID, time.Duration, int, error) {
	startTime := time.Now()
	resp, err := j.client.CreateChatCompletion(ctx, j.req)
	duration := time.Since(startTime)
	if err != nil {
		return nil, duration, 0, fmt.Errorf("GenerateContent error: %w", err)
	}

	totalTokens := 0
	if resp.Usage != nil {
		totalTokens = resp.Usage.TotalTokens
	}

	responses := []llms.LlmResposeWithChatID{}
	for _, choice := range resp.Choices {
		message := jambaClient.Message{
			Role:      jambaClient.RoleAssistant,
			Content:   "No content provided by the assistant",
			ToolCalls: choice.Message.ToolCalls,
		}
		if choice.Message.Content != nil {
			message.Content = *choice.Message.Content
		}
		j.req.Messages = append(j.req.Messages, message)
		if len(choice.Message.ToolCalls) == 0 && choice.Message.Content != nil {
			llmResponse := llms.RoomsResp{}
			err = json.Unmarshal([]byte(*choice.Message.Content), &llmResponse)
			if err != nil {
				return nil, duration, totalTokens, fmt.Errorf("failed to unmarshal response: %w", err)
			}

			responses = append(responses, llms.LlmResposeWithChatID{
				RoomsResp: llmResponse,
			})

			continue
		}

		for _, toolCall := range choice.Message.ToolCalls {
			var llmResponse interface{} = &llms.RoomsResp{}
			if toolCall.Function.Name == llms.URLsName {
				llmResponse = &llms.UrlsResp{}
			}

			err = json.Unmarshal([]byte(toolCall.Function.Arguments), &llmResponse)
			if err != nil {
				return nil, duration, totalTokens, fmt.Errorf("failed to unmarshal response: %w", err)
			}

			result := llms.LlmResposeWithChatID{
				ChatID:   toolCall.ID,
				ToolName: toolCall.Function.Name,
			}

			if urls, ok := llmResponse.(*llms.UrlsResp); ok {
				result.URLs = urls.URLs
			} else if rooms, ok := llmResponse.(*llms.RoomsResp); ok {
				result.Rooms = rooms.Rooms
			}

			responses = append(responses, result)
		}
	}

	return responses, duration, totalTokens, nil
}

func (j *jamba) ModelName() string {
	return j.req.Model
}

func (j *jamba) ImageSupport() bool {
	return j.imageSupport
}

func (j *jamba) ResetChat() {
	j.req.Messages = nil
}

func (j *jamba) Guided(mode bool) {
	j.guided = mode
}

func (j *jamba) RoomToolOnly() {
}
