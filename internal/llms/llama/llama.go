package llama

// import (
// 	"context"
// 	"encoding/json"
// 	"time"

// 	"github.com/martinbockt/esc-llm-webscraper/internal/llms"
// 	"github.com/martinbockt/esc-llm-webscraper/pkg/togetherai"
// )

// var _ = (llms.Plugin)(&llama{})

// type llama struct {
// 	client       *togetherai.ChatService
// 	req          togetherai.ChatCompletionRequest
// 	imageSupport bool
// 	guided       bool
// }

// func New(togAIClient *togetherai.ChatService, modelName string, _ float32, imageSupport bool) llms.Plugin {
// 	req := togetherai.ChatCompletionRequest{
// 		Model:      modelName,
// 		ToolChoice: "required",
// 		MaxTokens:  8000,
// 		Tools: []togetherai.Tool{
// 			{
// 				Type: "function",
// 				Function: togetherai.Function{
// 					Name:        llms.RoomsName,
// 					Description: llms.RoomsDescription,
// 					Parameters:  generateSchemaMap(llms.RoomsResp{}),
// 				},
// 			},
// 			{
// 				Type: "function",
// 				Function: togetherai.Function{
// 					Name:        llms.URLsName,
// 					Description: llms.URLsDescription,
// 					Parameters:  generateSchemaMap(llms.UrlsResp{}),
// 				},
// 			},
// 		},
// 	}

// 	return &llama{
// 		client:       togAIClient,
// 		req:          req,
// 		imageSupport: imageSupport,
// 	}
// }

// func (l *llama) AddPrompt(_ []byte, text, _ string, toolName string) {
// 	message := togetherai.Message{
// 		Role:    togetherai.User,
// 		Content: text,
// 	}

// 	if toolName != "" {
// 		message.Role = togetherai.System
// 	}

// 	l.req.Messages = append(l.req.Messages, message)
// }

// func (l *llama) ExecutePrompt(ctx context.Context) ([]llms.LlmResposeWithChatID, time.Duration, error) {
// 	startTime := time.Now()
// 	resp, err := l.client.CreateChatCompletion(ctx, l.req)
// 	if err != nil {
// 		return nil, time.Duration(0), err
// 	}
// 	duration := time.Since(startTime)

// 	responses := []llms.LlmResposeWithChatID{}
// 	for _, choice := range resp.Choices {
// 		for _, toolCall := range choice.Message.ToolCalls {
// 			singleResponse := llms.LlmResponse{}
// 			err = json.Unmarshal([]byte(toolCall.Function.Arguments), &singleResponse)
// 			if err != nil {
// 				return nil, time.Duration(0), err
// 			}

// 			responses = append(responses, llms.LlmResposeWithChatID{
// 				LlmResponse: singleResponse,
// 				ChatID:      toolCall.ID,
// 				ToolName:    toolCall.Function.Name,
// 			})
// 		}
// 	}

// 	return responses, duration, nil
// }

// func (l *llama) ModelName() string {
// 	return l.req.Model
// }

// func (l *llama) ImageSupport() bool {
// 	return l.imageSupport
// }

// func (l *llama) ResetChat() {
// 	l.req.Messages = nil
// }

// func (l *llama) Guided(mode bool) {
// 	l.guided = mode
// }

// func (l *llama) RoomToolOnly() {
// }
