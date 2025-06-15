# 🕷️ Go Web Crawler

A lightweight, concurrent web crawler written in Go. It starts from a root URL, crawls internal links up to a specified limit, and saves the extracted content and titles into a JSON file.

## 🚀 Features

-   Concurrent HTTP fetching
-   HTML parsing with link discovery
-   Extracts page titles and body content
-   URL deduplication to avoid redundant crawling
-   Saves results to a pretty-printed JSON file

## 📦 Requirements

-   [Go 1.18+](https://golang.org/dl/)

## 🛠️ Installation

Clone the repository:

```bash
git clone https://github.com/yourusername/go-web-crawler.git
cd go-web-crawler
```

## Build the project:

```bash
go build -o crawler main.go
```

## ⚙️ Usage

```bash
./crawler -root="<starting-url>" -limit=<max-pages> -output="<output-filename>"
```

Example

```bash
./crawler -root="https://web-scraping.dev/" -limit=50 -output="results.json"
```

Flags
Flag Description Default
-root Starting URL for the crawler https://web-scraping.dev/
-limit Maximum number of pages to crawl 50
-output Output file for the crawl results (JSON) results.json
🧠 How It Works

    Starts crawling from the root URL.

    Extracts all internal links and enqueues uncrawled ones.

    Extracts page <title> and up to 500 characters from the <body>.

    Stores each page’s title, URL, and content in a JSON array.

    Stops crawling when it hits the crawl limit or queue is empty.

🗂️ Output Format

results.json contains an array of objects like:

{
"Title": "Sample Page Title",
"Content": "First few hundred characters of body content...",
"Url": "https://example.com/page"
}
