package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
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
	size  int
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

func (q *Queue) Enqueue(url string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.size++
	q.members = append(q.members, url)
	return nil
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


func(crawl *CrawlSet) CrawlAdd( key string) {
	crawl.mu.Lock()
	defer crawl.mu.Unlock()
	if !crawl.crawl[key] {
		crawl.crawl[key] = true
		crawl.size++
	}
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
func parse(q *Queue, content []byte, db *db, currUrl string, c *CrawlSet) {
	z := html.NewTokenizer(bytes.NewReader(content))
	body := false
	tokenCount := 0
	pageContentLength := 0
	webpage := webpage{Url: currUrl, Title: "", Content: ""}
	
	for {
		tt := z.Next()
		if tt == html.ErrorToken || tokenCount > 500 {
			db.content = append(db.content, webpage)
			break
		}
		token := z.Token()
		if token.Type == html.StartTagToken {
			if token.Data == "javascript" {
				z.Next()
				continue
			}
			if token.Data == "body" {
				body = true
			}
			if token.Data == "title" {
				z.Next()
				webpage.Title = z.Token().Data
			}
			if token.Data == "a" {
				for _, v := range token.Attr {
					if v.Key == "href" {
						c.mu.Lock()
						_, ok := c.crawl[v.Val]
						c.mu.Unlock()
						if !ok {
							q.Enqueue(v.Val)
							c.CrawlAdd( v.Val)
						}
					}
				}
			}
		}
		if token.Type == html.TextToken && body && pageContentLength < 500 {
			webpage.Content += token.Data
			pageContentLength += len(token.Data)
		}
		tokenCount++
	}
}

func main() {
	root := flag.String("root","https://web-scraping.dev/", "Starting Url")
	c := make(chan []byte, 10)
	limit := flag.Int("limit" , 50, "Number of pages to be crawled")
	output := flag.String("output", "results.json","output file name")
	q := Queue{size: 0, members: []string{},crawled: 0}
	crawler := CrawlSet{crawl: make(map[string]bool)}
	database := db{}
	flag.Parse()
	t0 := time.Now()
	q.Enqueue(*root)
	crawler.CrawlAdd(*root)

	for q.size > 0 && q.crawled < *limit {
		url, err := q.Dequeue()
		if err != nil {
			break
		}
		go fetch(url, c)
		data := <-c
		if len(data) == 0 {
			continue
		}
		parse(&q, data, &database, url, &crawler)
		
		q.crawled++
	}
	err := saveToJSON(&database, *output)
	if err != nil {
		fmt.Println("Failed to save JSON:", err)
	} else {
		fmt.Println("Saved crawl results to",*output)
	}
	fmt.Println("Time taken to crawl:" ,time.Since(t0).Seconds())
	fmt.Printf("Total pages crawled: %d\n", len(database.content))
	fmt.Printf("Total unique URLs seen: %d\n", crawler.size)
	fmt.Printf("Remaining in queue: %d\n", q.size)
	
}