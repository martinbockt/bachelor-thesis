package scraper_test

import (
	"os"
	"testing"

	"github.com/martinbockt/esc-llm-webscraper/internal/scraper"
	"go.uber.org/zap"
)

func TestWebsite(t *testing.T) {
	log, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Error creating logger: %v", err)
	}

	s, err := scraper.New(log, "", "", "", "", "", "")
	if err != nil {
		t.Fatalf("Error creating scraper: %v", err)
	}

	p, err := s.CreatePage()

	err = p.Navigate("https://alfsee-escape.de/indoor-escape-room/")
	if err != nil {
		t.Fatalf("Error navigating to URL: %v", err)
	}

	content, _, _, err := p.PageContent()
	if err != nil {
		t.Fatalf("Error getting page content: %v", err)
	}

	file, err := os.Create("output.html")
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		t.Fatalf("Error writing to file: %v", err)
	}

	t.Log("Website content successfully written to output.html")
}
