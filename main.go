package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"golang.org/x/net/html"
)

type Queue struct {
	size    int
	members []string
	mu      sync.Mutex
	crawled int
}

type CrawlSet struct {
	crawl map[string]bool
	mu    sync.Mutex
	size int
}

type db struct {
	content []webpage
	length  int
}

type webpage struct {
	Title   string
	Content string
	Url     string
}

func (q *Queue) Dequeue() (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.size == 0 {
		return "", errors.New("Queue Empty")
	}
	url := q.members[0]
	q.members = q.members[1:]
	q.size--
	return url, nil
}

func (q *Queue) Enqueue(url string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.size++
	q.members = append(q.members, url)
}

func (c *CrawlSet) CrawlAdd(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.crawl[key] {
		c.crawl[key] = true
		c.size++
	}
}

func saveToJSON(db *db, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(db.content)
}

func fetch(url string, c chan []byte) {
	resp, err := http.Get(url)
	if err != nil {
		c <- []byte{}
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c <- []byte{}
		return
	}
	c <- body
}

func parse(q *Queue, content []byte, db *db, currUrl string, c *CrawlSet, rootHost string) {
	z := html.NewTokenizer(bytes.NewReader(content))
	inBody := false
	tokenCount := 0
	contentLength := 0
	page := webpage{Url: currUrl}

	for {
		tt := z.Next()
		if tt == html.ErrorToken || tokenCount > 500 {
			db.content = append(db.content, page)
			break
		}

		tok := z.Token()
		if tok.Type == html.StartTagToken {
			switch tok.Data {
			case "title":
				z.Next()
				page.Title = z.Token().Data
			case "body":
				inBody = true
			case "a":
				for _, attr := range tok.Attr {
					if attr.Key == "href" {
						link, err := url.Parse(attr.Val)
						if err != nil {
							continue
						}
						if link.Host == "" || link.Host == rootHost {
							absUrl := link.String()
							c.mu.Lock()
							_, seen := c.crawl[absUrl]
							c.mu.Unlock()
							if !seen {
								q.Enqueue(absUrl)
								c.CrawlAdd(absUrl)
							}
						}
					}
				}
			}
		}

		if tok.Type == html.TextToken && inBody && contentLength < 500 {
			page.Content += tok.Data
			contentLength += len(tok.Data)
		}

		tokenCount++
	}
}

func main() {
	root := flag.String("root", "https://web-scraping.dev/", "Starting URL")
	limit := flag.Int("limit", 50, "Number of pages to crawl")
	output := flag.String("output", "results.json", "Output file name")
	verbose := flag.Bool("v", false, "Enable verbose logging")
	flag.Parse()

	parsedRoot, err := url.Parse(*root)
	if err != nil {
		fmt.Println("Invalid root URL")
		return
	}

	c := make(chan []byte, 10)
	q := Queue{}
	crawler := CrawlSet{crawl: make(map[string]bool)}
	database := db{}

	t0 := time.Now()
	q.Enqueue(*root)
	crawler.CrawlAdd(*root)

	for q.size > 0 && q.crawled < *limit {
		currUrl, err := q.Dequeue()
		if err != nil {
			break
		}

		if *verbose {
			fmt.Println("Crawling:", currUrl)
		}

		go fetch(currUrl, c)
		data := <-c
		if len(data) == 0 {
			continue
		}

		parse(&q, data, &database, currUrl, &crawler, parsedRoot.Host)
		q.crawled++
	}

	err = saveToJSON(&database, *output)
	if err != nil {
		fmt.Println("Failed to save JSON:", err)
	} else {
		fmt.Println("Saved crawl results to", *output)
	}

	fmt.Println("Time taken to crawl:", time.Since(t0).Seconds(), "seconds")
	fmt.Printf("Total pages crawled: %d\n", len(database.content))
	fmt.Printf("Total unique URLs seen: %d\n", crawler.size)
	fmt.Printf("Remaining in queue: %d\n", q.size)
}
