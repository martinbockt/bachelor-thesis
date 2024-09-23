package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"time"

	"cloud.google.com/go/vertexai/genai"
	config "github.com/martinbockt/esc-llm-webscraper/cmd/api/internal"
	"github.com/martinbockt/esc-llm-webscraper/internal/llms"
	"github.com/martinbockt/esc-llm-webscraper/internal/llms/gpt"
	"github.com/martinbockt/esc-llm-webscraper/internal/llms/jamba"
	"github.com/martinbockt/esc-llm-webscraper/internal/llms/mistral"
	"github.com/martinbockt/esc-llm-webscraper/internal/llms/vertex"
	"github.com/martinbockt/esc-llm-webscraper/internal/output"
	"github.com/martinbockt/esc-llm-webscraper/internal/scraper"
	"github.com/martinbockt/esc-llm-webscraper/pkg/jambaClient"
	"github.com/martinbockt/esc-llm-webscraper/pkg/togetherai"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

func main() {
	logger, err := newLogger()
	if err != nil {
		panic(err)
	}
	cfg, err := config.New()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}
	ctx := context.Background()
	vertexClient, err := genai.NewClient(ctx, cfg.GCloudProjectID, cfg.GCloudLocationID)
	if err != nil {
		logger.Fatal("failed to init genAI client", zap.Error(err))
	}
	gptClient := openai.NewClient(cfg.ChatGPTToken)
	togetheraiClient := togetherai.NewChatService(logger, cfg.TogetherAIToken)
	jClient := jambaClient.NewChatService(logger, cfg.JambaToken)

	llmList := initLLMs(vertexClient, gptClient, togetheraiClient, jClient, cfg.ClaudeToken, cfg.MistralToken)

	scraper, err := scraper.New(logger, cfg.ProxyServer, cfg.ProxyUsername, cfg.ProxyPassword, cfg.LoginEmail, cfg.LoginPassword, cfg.OTPSecret)
	if err != nil {
		logger.Fatal("failed to init scraper", zap.Error(err))
	}

	err = run(ctx, logger, cfg, llmList, scraper)
	if err != nil {
		logger.Fatal("failed to run", zap.Error(err))
	}
}

func newLogger() (*zap.Logger, error) {
	logFilePath := "./log/errors.log"
	if err := os.MkdirAll("./log", 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	file.Close()

	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{
		logFilePath,
	}

	logger, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	logger.Info("Logger initialized successfully", zap.String("logFilePath", logFilePath))

	return logger, nil
}

func initLLMs(vertexClient *genai.Client, gptClient *openai.Client, togetheraiClient *togetherai.ChatService, jambaClient *jambaClient.ChatService, claudeToken, mistralToken string) *llms.Registry {
	llmRegistry := llms.NewRegistry()
	llmRegistry.Register(jamba.New(jambaClient, "jamba-1.5-large", 0, false))
	llmRegistry.Register(jamba.New(jambaClient, "jamba-1.5-mini", 0, false))

	// llmRegistry.Register(llama.New(togetheraiClient, "meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo", 0, false))
	// llmRegistry.Register(mistral.New("mistral-large-2407", mistralToken, false))
	llmRegistry.Register(mistral.New("mistral-large-2407", mistralToken, false))
	llmRegistry.Register(vertex.New(vertexClient, "gemini-1.5-flash-001", 0.5, false))
	llmRegistry.Register(vertex.New(vertexClient, "gemini-1.5-pro-001", 0.5, false))

	// llmRegistry.Register(claude.New("claude-3-5-sonnet-20240620", claudeToken, true))
	// llmRegistry.Register(gpt.New(gptClient, openai.GPT4o, true))
	llmRegistry.Register(gpt.New(gptClient, openai.GPT4oMini, true))

	return llmRegistry
}

func run(ctx context.Context, logger *zap.Logger, cfg *config.Config, llmList *llms.Registry, scraperBrowser scraper.ScraperBrowser) error {
	var errs error
	rooms, errs := parseEscapeRooms("./escapeRooms.json")
	if errs != nil {
		return fmt.Errorf("failed to parse escape rooms: %w", errs)
	}

	var errorMutex sync.Mutex
	syncGroup := sync.WaitGroup{}

	for _, llm := range llmList.Plugins() {
		syncGroup.Add(1)
		go func() {
			defer syncGroup.Done()
			scraper, err := scraperBrowser.CreatePage()
			if err != nil {
				logger.Error("failed to create page", zap.Error(err))

				return
			}
			op := output.New()
			existingRooms, err := op.ReadOutputCSV(llm.ModelName())
			if err != nil {
				logger.Error("failed to read existing rooms", zap.Error(err))

				return
			}

			for index, room := range rooms {
				llm.Guided(false)
				if slices.ContainsFunc(existingRooms, func(eRoom output.Information) bool {
					return eRoom.RoomName == room.Name
				}) {
					logger.Info("room already scraped", zap.String("room", room.Name))

					continue
				}

				var err error
				logger.Info("loaded llm", zap.String("name", llm.ModelName()))
				prompt := "List all escape rooms of the website. If there is, use the Escape Room detail pages as content source."
				rooms := []llms.Room{}
				response := []llms.LlmResposeWithChatID{
					{
						UrlsResp: llms.UrlsResp{URLs: []string{room.URL}},
						ChatID:   "",
					},
				}
				var done bool
				startTime := time.Now()
				llmDuration := time.Duration(0)
				var tokenCount int
				var websiteLength, shortenedLength, websitesChecked int
				var websiteContent []string
				for i := range cfg.Limit {
					done = true
					for _, resp := range response {
						rooms = append(rooms, resp.Rooms...)

						var websiteMaxLength, shortLength int
						if len(resp.URLs) != 0 {
							done = false
						} else {
							llm.AddPrompt(nil, "added", resp.ChatID, resp.ToolName)

							continue
						}

						for _, url := range resp.URLs {
							var content string
							err = scraper.Navigate(url)
							if err != nil {
								err = fmt.Errorf("failed to navigate: %w", err)

								break
							}

							content, websiteMaxLength, shortLength, err = scraper.PageContent()
							websiteLength += websiteMaxLength
							shortenedLength += shortLength
							if err != nil {
								err = fmt.Errorf("failed to get page content: %w", err)

								break
							}

							logger.Info("page content length", zap.Int("initial length", websiteLength), zap.Int("shortened length", shortenedLength))
							p := fmt.Sprintf("Current URL: %s; Current website content: %s", url, content)
							prompt += p
							websiteContent = append(websiteContent, p)
							websitesChecked++
						}
						llm.AddPrompt(nil, prompt, resp.ChatID, resp.ToolName)
					}
					if done || err != nil || len(response) == 0 {
						addToOutput(op,
							output.Information{
								ID:                   index,
								LLM:                  llm.ModelName(),
								LLMDuration:          llmDuration,
								RequestDuration:      time.Since(startTime),
								WebsitesChecked:      websitesChecked,
								WebsiteMaxLength:     websiteLength,
								WebsiteReducedLength: shortenedLength,
								ProviderURL:          room.URL,
								ProviderName:         room.Name,
								TokenLimitReached:    false,
								TokenCount:           tokenCount,
							},
							rooms,
							err)
						logger.Info("done", zap.Bool("done", done), zap.Error(err))
						llm.ResetChat()

						break
					}

					result, duration, reqTokenCount, err := llm.ExecutePrompt(ctx)
					llmDuration += duration
					tokenCount = reqTokenCount
					if err != nil {
						logger.Error("failed to prompt", zap.Error(err))
					}
					logger.Info("result", zap.Any("result", result))
					response = result
					if err != nil || i == cfg.Limit {
						llm.ResetChat()
						tokenLimit := false
						if len(rooms) == 0 {
							tokenLimit = true
							// we assume error means token limit reached
							llm.RoomToolOnly()
							for i, wc := range websiteContent {
								if i == 0 {
									continue // skip the first one / main page
								}

								llm.RoomToolOnly()
								llm.AddPrompt(nil, wc, "", "")
								resp, time, _, err := llm.ExecutePrompt(ctx)
								llm.ResetChat()
								if err != nil {
									logger.Error("failed to execute prompt:", zap.Error(err))

									break
								}

								llmDuration += time
								for _, res := range resp {
									rooms = append(rooms, res.Rooms...)
								}
							}
						}
						addToOutput(op,
							output.Information{
								ID:                   index,
								LLM:                  llm.ModelName(),
								LLMDuration:          llmDuration,
								RequestDuration:      time.Since(startTime),
								WebsitesChecked:      websitesChecked,
								WebsiteMaxLength:     websiteLength,
								WebsiteReducedLength: shortenedLength,
								ProviderURL:          room.URL,
								ProviderName:         room.Name,
								TokenLimitReached:    tokenLimit,
								TokenCount:           tokenCount,
							},
							rooms,
							err)
						logger.Error("failed to execute prompt", zap.Error(err), zap.Int("limit", i))

						break
					}
				}
				errs = errors.Join(errs, err)
			}
			logger.Info("rooms", zap.Any("rooms", rooms))
			err = op.SaveAsCSV(llm.ModelName())

			errorMutex.Lock()
			errs = errors.Join(errs, err)
			errorMutex.Unlock()
		}()
	}
	syncGroup.Wait()

	return errs
}

func addToOutput(op *output.Output, inf output.Information, rooms []llms.Room, err error) {
	if err != nil {
		inf.Error = err.Error()
	}

	if len(rooms) == 0 {
		op.AddInformation(inf)
	}
	for _, room := range rooms {
		inf.RoomName = room.Name
		inf.Description = room.Description
		inf.MinPlayers = room.PlayersMin
		inf.MaxPlayers = room.PlayersMax
		inf.Duration = room.Duration
		inf.BookingURL = room.BookingURL
		inf.DetailPageURL = room.DetailPageURL
		inf.ImageURL = room.ImageURL
		inf.Genre = room.Genre
		inf.Difficulty = room.Difficulty
		op.AddInformation(inf)
	}
}
