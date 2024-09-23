// nolint: tagalign
package llms

import (
	"context"
	"time"
)

type URL string // URL is a string that represents a URL.

const (
	NavigateToURL URL = "navigate_to_url"
	Done          URL = "done"
)

type LlmResposeWithChatID struct {
	ChatID     string
	ToolName   string
	TokenUsage int
	UrlsResp
	RoomsResp
}

type LlmResponse struct {
	Action        URL    `json:"action"         description:"Action to take, either navigate to a URL to receive more content or done to finish the conversation." enum:"navigate_to_url,done" required:"true"`
	URL           string `json:"url"            description:"URL to navigate to. Only used if action is navigate_to_url. The URL must be a valid URL."`
	PartialResult []Room `json:"partial_result" description:"Array of partial results. Only fill if you scraped the detail page of an escape room."`
}

type UrlsResp struct {
	URLs []string `json:"urls" description:"Array of URLs to scrape (most relevant are escape room detail pages). The URLs must be valid URLs."`
}

type RoomsResp struct {
	Rooms []Room `json:"rooms" description:"Array of escape rooms. The escape room data needs to be copied and formatted from the website."`
}

type Room struct {
	Name          string `json:"name"            description:"Name of the escape room"`
	Description   string `json:"description"     description:"Description of the escape room. You find it on the detail page of the escape room."`
	PlayersMin    int    `json:"players_min"     description:"Minimum number of players"`
	PlayersMax    int    `json:"players_max"     description:"Maximum number of players"`
	Duration      int    `json:"duration"        description:"Duration of the escape room in minutes"`
	BookingURL    string `json:"booking_url"     description:"The full URL/a link to book the escape room."`
	DetailPageURL string `json:"detail_page_url" description:"The full URL/a link to the detail page of the escape room"`
	ImageURL      string `json:"image_url"       description:"The full URL/a link to a room related image. Most of the time on top of the detail page of an escape room."`
	Genre         string `json:"genre"           description:"Select the genre/enum value that most closely matches the escape room."                                     enum:"Adventure,Crime,Egypt,Fantasy,Historical,Horror,Medieval,Prison,Science Fiction,Steampunk,Western" required:"true"`
	Difficulty    string `json:"difficulty"      description:"Difficulty of the escape room"`
}

const (
	SystemPrompt     = "You are a website parser bot. You can interact with the websites provided and retrieve content from them. You can navigate to a URL to get more content or end the conversation. Your goal is to provide as much information from the website as possible about the requested topic."
	URLsName         = "more_content"
	URLsDescription  = "For more content provide the URLs of the website. Most likely, call this first to get the content of the detail pages."
	RoomsName        = "list_escape_rooms"
	RoomsDescription = "List all available escape rooms of the website. With this you are ending the conversation."
)

type Plugin interface {
	ModelName() string
	AddPrompt(image []byte, text, chatID, toolName string)
	ExecutePrompt(ctx context.Context) ([]LlmResposeWithChatID, time.Duration, int, error)
	ImageSupport() bool
	ResetChat()
	Guided(mode bool)
	RoomToolOnly()
}

type Registry struct {
	plugins map[string]Plugin
}

func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
	}
}

func (r *Registry) Register(p Plugin) {
	r.plugins[p.ModelName()] = p
}

func (r *Registry) Plugin(modelName string) Plugin {
	return r.plugins[modelName]
}

func (r *Registry) Plugins() []Plugin {
	var plugins = make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}
