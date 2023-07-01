package main

import (
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

const (
	BaseURL        = "https://www.eenadu.net"
	outputFileName = "eenadu.json"
)

type Article struct {
	URL           string `json:"url"`
	Title         string `json:"title"`
	DatePublished string `json:"date_published"`
	Content       string `json:"content"`
}

type Job struct {
	URL   string
	Depth int
}

var (
	visited = make(map[string]bool)
	queue   chan Job
	wg      sync.WaitGroup
	lock    sync.Mutex
)

func main() {
	queue = make(chan Job)
	wg.Add(1)

	go func() {
		defer wg.Done()
		for job := range queue {
			processURL(job)
		}
	}()

	queue <- Job{URL: BaseURL, Depth: 0}
	wg.Wait()
	close(queue)
}

func processURL(job Job) {
	url := job.URL
	depth := job.Depth

	if depth > 2 {
		return
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to get URL %s: %v", url, err)
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("Failed to parse document for URL %s: %v", url, err)
		return
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

	article := &Article{
		URL:           url,
		Title:         title,
		DatePublished: datePublished,
		Content:       content,
	}

	writeArticleToFile(article, outputFileName)

	lock.Lock()
	for _, link := range urls {
		if !visited[link] {
			visited[link] = true
			wg.Add(1)
			go func(link string) {
				defer wg.Done()
				queue <- Job{URL: link, Depth: depth + 1}
			}(link)
		}
	}
	lock.Unlock()
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
