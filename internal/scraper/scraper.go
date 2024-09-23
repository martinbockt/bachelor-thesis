package scraper

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/stealth"
	"go.uber.org/zap"
)

type Scraper struct {
	log                   *zap.Logger
	defaultBrowserTimeout time.Duration
	browser               *rod.Browser
	page                  *rod.Page
	loginEmail            string
	loginPassword         string
	oTPSecret             string
}

func (s *Scraper) getPage() *rod.Page {
	return s.page.Timeout(s.defaultBrowserTimeout)
}

type ScraperBrowser interface {
	CreatePage() (ScraperPage, error)
}

type ScraperPage interface {
	ClickButton(selector string) error
	EnterInput(selector, input string) error
	PageContent() (string, int, int, error)
	Navigate(url string) error
	GetScreenshot() ([]byte, error)
}

func New(log *zap.Logger, proxyServer, proxyUsername, proxyPassword, loginEmail, loginPassword, oTPSecret string) (ScraperBrowser, error) {
	page, err := newBrowser(log, proxyServer, proxyUsername, proxyPassword)
	if err != nil {
		return nil, err
	}

	return &Scraper{
		log:                   log,
		browser:               page,
		loginEmail:            loginEmail,
		loginPassword:         loginPassword,
		oTPSecret:             oTPSecret,
		defaultBrowserTimeout: 10 * time.Second,
	}, nil
}

func (s *Scraper) CreatePage() (ScraperPage, error) {
	s.log.Info("starting stealth page")
	page, err := stealth.Page(s.browser)
	if err != nil {
		return nil, fmt.Errorf("failed to create stealth-page: %w", err)
	}

	s.log.Info("setting fullscreen")
	page = page.MustWindowFullscreen()

	scraper := *s
	scraper.page = page

	return &scraper, nil
}

func newBrowser(log *zap.Logger, proxyServer string, proxyUsername string, proxyPassword string) (*rod.Browser, error) {
	log.Info("creating new browser")
	path, _ := launcher.LookPath()
	launcher := launcher.
		New().
		Headless(true).
		Bin(path).
		NoSandbox(true)

	if proxyServer != "" && proxyUsername != "" && proxyPassword != "" {
		launcher.Set(flags.ProxyServer, proxyServer)
	}

	log.With(zap.Any("launcher-flags", launcher.Flags)).Info("launching browser")
	u, err := launcher.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch new browser: %w", err)
	}

	log.Info("connecting to browser")
	browser := rod.New().ControlURL(u)
	err = browser.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	if proxyServer != "" && proxyUsername != "" && proxyPassword != "" {
		log.Info("setting up auth handler")
		go func() {
			err = browser.HandleAuth(proxyUsername, proxyPassword)()
			if err != nil {
				log.Error("failed to handle auth for proxy", zap.Error(err))
			}
		}()
	}

	return browser, nil
}
