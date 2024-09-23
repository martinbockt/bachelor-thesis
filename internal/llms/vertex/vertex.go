package vertex

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/martinbockt/esc-llm-webscraper/internal/llms"
)

var _ = (llms.Plugin)(&vertex{})

type vertex struct {
	client       *genai.Client
	model        *genai.GenerativeModel
	messages     []genai.Part
	chatSession  *genai.ChatSession
	guided       bool
	imageSupport bool
}

func New(client *genai.Client, modelName string, temperature float32, imageSupport bool) llms.Plugin {
	model := client.GenerativeModel(modelName)
	model.SetTemperature(temperature)
	model.ToolConfig = &genai.ToolConfig{
		FunctionCallingConfig: &genai.FunctionCallingConfig{
			Mode:                 genai.FunctionCallingAny,
			AllowedFunctionNames: []string{llms.URLsName, llms.RoomsName},
		},
	}

	model.Tools = []*genai.Tool{
		{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:        llms.URLsName,
					Description: llms.URLsDescription,
					Parameters:  generateSchema(llms.UrlsResp{}),
				},
				{
					Name:        llms.RoomsName,
					Description: llms.RoomsDescription,
					Parameters:  generateSchema(llms.RoomsResp{}),
				},
			},
		},
	}

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(llms.SystemPrompt)},
	}

	return &vertex{
		client:       client,
		model:        model,
		imageSupport: imageSupport,
	}
}

func (v *vertex) AddPrompt(image []byte, text, _, functionName string) {
	// img := genai.ImageData("webp", image)
	if functionName == "" {
		v.chatSession = v.model.StartChat()
	}

	if v.guided {
		v.model.ToolConfig.FunctionCallingConfig.AllowedFunctionNames = []string{llms.URLsName}
	}

	genAIPart := []genai.Part{
		// img,
		genai.Text(text),
	}
	if functionName != "" {
		v.model.ToolConfig.FunctionCallingConfig.AllowedFunctionNames = []string{llms.URLsName, llms.RoomsName}
		genAIPart = []genai.Part{
			genai.FunctionResponse{
				Name:     functionName,
				Response: map[string]any{"content": text},
			},
		}
	}

	v.messages = genAIPart
}

func (v *vertex) ExecutePrompt(ctx context.Context) ([]llms.LlmResposeWithChatID, time.Duration, int, error) {
	startTime := time.Now()
	resp, err := v.chatSession.SendMessage(ctx, v.messages...)
	duration := time.Since(startTime)
	if err != nil {
		return nil, duration, 0, fmt.Errorf("GenerateContent error: %w", err)
	}

	tokenCount := 0
	if resp.UsageMetadata != nil {
		tokenCount = int(resp.UsageMetadata.TotalTokenCount)
	}

	result := []llms.LlmResposeWithChatID{}
	for _, part := range resp.Candidates {
		for _, fCall := range part.FunctionCalls() {
			var args interface{} = &llms.RoomsResp{}
			if fCall.Name == llms.URLsName {
				args = &llms.UrlsResp{}
			}

			jsonArg, err := json.Marshal(fCall.Args)
			if err != nil {
				return nil, time.Duration(0), tokenCount, fmt.Errorf("failed to marshal arg: %w", err)
			}
			err = json.Unmarshal(jsonArg, args)
			if err != nil {
				return nil, time.Duration(0), tokenCount, fmt.Errorf("failed to unmarshal arg: %w", err)
			}

			resp := llms.LlmResposeWithChatID{
				ToolName: fCall.Name,
			}

			if urls, ok := args.(*llms.UrlsResp); ok {
				resp.URLs = urls.URLs
			} else if rooms, ok := args.(*llms.RoomsResp); ok {
				resp.Rooms = rooms.Rooms
			}

			result = append(result, resp)
		}
	}

	return result, duration, tokenCount, nil
}

func (v *vertex) ModelName() string {
	return v.model.Name()
}

func (v *vertex) ImageSupport() bool {
	return v.imageSupport
}

func (v *vertex) ResetChat() {
	v.model.ToolConfig.FunctionCallingConfig.AllowedFunctionNames = []string{llms.URLsName, llms.RoomsName}
	v.chatSession.History = nil
	v.messages = nil
}

func (v *vertex) Guided(mode bool) {
	v.guided = mode
}

func (v *vertex) RoomToolOnly() {
}
