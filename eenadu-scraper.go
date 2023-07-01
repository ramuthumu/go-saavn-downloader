package main

import (
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	BaseURL       = "https://www.eenadu.net"
	outputFileName = "eenadu.json"
)

type Article struct {
	URL            string `json:"url"`
	Title          string `json:"title"`
	DatePublished  string `json:"date_published"`
	Content        string `json:"content"`
}

func main() {
	visited := make(map[string]bool)
	queue := []string{BaseURL}

	for len(queue) > 0 {
		currentURL := queue[0]
		queue = queue[1:]

		if visited[currentURL] {
			continue
		}
		visited[currentURL] = true

		article, urls, err := extractContent(currentURL)
		if err != nil {
			log.Printf("Failed to extract content from %s: %v", currentURL, err)
			continue
		}

		writeArticleToFile(article, outputFileName)

		for _, url := range urls {
			if !visited[url] {
				queue = append(queue, url)
			}
		}
	}
}

func extractContent(url string) (*Article, []string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	title := doc.Find("h1").Text()
	datePublished := strings.TrimSpace(doc.Find(".pub-t").Text())
	contentSelection := doc.Find(".fullstory p")
	content := strings.Join(contentSelection.Map(func(_ int, s *goquery.Selection) string {
		return s.Text()
	}), "\n\n")

	var urls []string
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && strings.HasPrefix(href, BaseURL) {
			urls = append(urls, href)
		}
	})

	return &Article{
		URL:           url,
		Title:         title,
		DatePublished: datePublished,
		Content:       content,
	}, urls, nil
}

func writeArticleToFile(article *Article, filename string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open file %s: %v", filename, err)
		return
	}
	defer file.Close()

	data, err := json.Marshal(article)
	if err != nil {
		log.Printf("Failed to marshal article: %v", err)
		return
	}

	_, err = file.Write(data)
	if err != nil {
		log.Printf("Failed to write to file: %v", err)
		return
	}

	_, err = file.Write([]byte("\n"))
	if err != nil {
		log.Printf("Failed to write to file: %v", err)
	}
}
