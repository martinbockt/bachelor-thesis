package scraper

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"golang.org/x/net/html"
)

func (s *Scraper) ClickButton(selector string) error {
	el, err := s.getPage().Element(selector)
	if err != nil {
		return fmt.Errorf("failed to find element: %w", err)
	}
	err = el.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return fmt.Errorf("failed to click element: %w", err)
	}

	return nil
}

func (s *Scraper) EnterInput(selector, input string) error {
	el, err := s.getPage().Element(selector)
	if err != nil {
		return fmt.Errorf("failed to find element: %w", err)
	}
	err = el.Input(input)
	if err != nil {
		return fmt.Errorf("failed to input text: %w", err)
	}

	return nil
}

func (s *Scraper) PageContent() (string, int, int, error) {
	page, err := s.getPage().HTML()
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to get html: %w", err)
	}

	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to parse html: %w", err)
	}

	// Find the <body> node
	bodyNode := findBodyNode(doc)
	if bodyNode == nil {
		return "", 0, 0, errors.New("failed to find body node")
	}

	// Clean up the HTML
	removeUnwantedTags(bodyNode, "script")
	removeUnwantedTags(bodyNode, "noscript")
	removeUnwantedTags(bodyNode, "style")
	removeUnwantedTags(bodyNode, "iframe")
	normalizeWhitespace(bodyNode)
	removeComments(bodyNode)
	removeAllAttributesExceptImportant(bodyNode)
	removeEmptyLinks(bodyNode)
	removeEmptyElements(bodyNode)

	var buf bytes.Buffer
	if err := html.Render(&buf, bodyNode); err != nil {
		return "", 0, 0, fmt.Errorf("failed to render html: %w", err)
	}

	return buf.String(), len(page), len(buf.String()), nil
}

func (s *Scraper) Navigate(url string) error {
	err := s.getPage().Navigate(url)
	if err != nil {
		return fmt.Errorf("failed to navigate to url: %s: %w", url, err)
	}
	err = s.getPage().WaitStable(time.Second)
	if err != nil {
		return fmt.Errorf("failed to wait for page to load: %w", err)
	}

	return nil
}

func (s *Scraper) GetScreenshot() ([]byte, error) {
	byes, err := s.getPage().Screenshot(true, &proto.PageCaptureScreenshot{
		Format:                proto.PageCaptureScreenshotFormatWebp,
		Quality:               createPointer(50),
		CaptureBeyondViewport: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to take screenshot: %w", err)
	}

	return byes, nil
}

func createPointer[A any](value A) *A {
	return &value
}
