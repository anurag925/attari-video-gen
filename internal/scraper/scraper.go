package scraper

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Config struct {
	UserAgent string
}

func Scrape(cfg Config, pageURL string) (string, error) {
	if cfg.UserAgent == "" {
		cfg.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
	}

	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	host := strings.ToLower(parsedURL.Host)

	var text string
	if strings.Contains(host, "reddit.com") {
		text, err = scrapeReddit(pageURL, cfg.UserAgent)
	} else {
		text, err = scrapeGeneric(pageURL, cfg.UserAgent)
	}

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(text), nil
}

func scrapeReddit(pageURL, userAgent string) (string, error) {
	// Convert old.reddit.com or www.reddit.com to reddit.com
	pageURL = strings.Replace(pageURL, "old.reddit.com", "www.reddit.com", 1)
	pageURL = strings.Replace(pageURL, "www.reddit.com", "reddit.com", 1)

	// Add .json to get structured data
	jsonURL := strings.Replace(pageURL, "/r/", "/r/.json?路=", 1)
	if strings.Contains(pageURL, ".json") {
		jsonURL = pageURL
	}

	req, err := http.NewRequest("GET", jsonURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Reddit page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Reddit returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse Reddit page: %w", err)
	}

	var sb strings.Builder

	// Find the post title
	title := doc.Find("h1").First().Text()
	if title != "" {
		sb.WriteString(title)
		sb.WriteString(". ")
	}

	// Find post body - Reddit specific selectors
	doc.Find("[data-testid='post-container']").Each(func(i int, s *goquery.Selection) {
		// Get the post body
		s.Find("[data-testid='post-text']").Each(func(j int, sel *goquery.Selection) {
			text := sel.Text()
			if text != "" {
				sb.WriteString(text)
				sb.WriteString(" ")
			}
		})
	})

	// Fallback: find main post content
	if sb.Len() == 0 {
		doc.Find(".richlink").Each(func(i int, s *goquery.Selection) {
			text := s.Text()
			if text != "" {
				sb.WriteString(text)
				sb.WriteString(" ")
			}
		})
	}

	// Another fallback for new Reddit
	if sb.Len() == 0 {
		doc.Find("shreddit-post").Each(func(i int, s *goquery.Selection) {
			text, _ := s.Attr("post-text")
			if text != "" {
				sb.WriteString(text)
				sb.WriteString(" ")
			}
		})
	}

	text := sb.String()
	if text == "" {
		return "", fmt.Errorf("could not extract text from Reddit post")
	}

	return text, nil
}

func scrapeGeneric(pageURL, userAgent string) (string, error) {
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("page returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse page: %w", err)
	}

	var sb strings.Builder

	// Remove script, style, nav, header, footer elements
	doc.Find("script, style, nav, header, footer, aside, .ads, .sidebar, .comments").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	// Try to find main content
	selections := []string{
		"article",
		"[role='main']",
		"main",
		".post-content",
		".entry",
		".content",
		"#content",
		".post",
		".story",
	}

	var content *goquery.Selection
	for _, sel := range selections {
		content = doc.Find(sel).First()
		if content.Length() > 0 {
			break
		}
	}

	if content == nil || content.Length() == 0 {
		content = doc.Selection
	}

	// Extract text
	content.Each(func(i int, s *goquery.Selection) {
		// Only process direct text nodes and paragraph-like elements
		tag := goquery.NodeName(s)
		if tag == "p" || tag == "div" || tag == "article" || tag == "section" || tag == "main" {
			text := strings.TrimSpace(s.Text())
			if text != "" && len(text) > 20 {
				sb.WriteString(text)
				sb.WriteString(" ")
			}
		}
	})

	// Fallback: get all paragraph text
	if sb.Len() < 50 {
		sb.Reset()
		doc.Find("p").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if text != "" && len(text) > 10 {
				sb.WriteString(text)
				sb.WriteString(" ")
			}
		})
	}

	text := sb.String()
	if text == "" {
		return "", fmt.Errorf("could not extract text from page")
	}

	return text, nil
}
