package config

import (
	"fmt"

	"github.com/alexflint/go-arg"
)

type Config struct {
	GCloudProjectID  string `arg:"--listen,env:GCLOUDPROJECTID"`
	GCloudLocationID string `arg:"--listen,env:GCLOUDLOCATIONID"`
	ChatGPTToken     string `arg:"--listen,env:CHATGPTTOKEN"`
	TogetherAIToken  string `arg:"--listen,env:TOGETHERAITOKEN"`
	ClaudeToken      string `arg:"--listen,env:CLAUDETOKEN"`
	MistralToken     string `arg:"--listen,env:MISTRALTOKEN"`
	JambaToken       string `arg:"--listen,env:JAMBATOKEN"`
	Limit            int    `arg:"--limit,env:LIMIT"`

	ProxyServer   string
	ProxyUsername string
	ProxyPassword string
	LoginEmail    string
	LoginPassword string
	OTPSecret     string
}

func New() (*Config, error) {
	c := &Config{
		Limit: 50,
	}

	err := arg.Parse(c) // nolint:typecheck
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return c, nil
}
