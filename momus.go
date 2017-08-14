// Package momus is a web scraper made to health check all the links (internal and external) inside a given site.
package momus

import (
	"golang.org/x/net/html"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// Link health check result
type LinkResult struct {
	Link       string
	StatusCode int
	rawLink    string
}

// Link health checker
type HealthChecker struct {
	onlyDeadLinks bool
}

var (
	mutex          sync.Mutex
	parsedStartUrl *url.URL
)

type Config struct {
	OnlyDeadLinks bool
}

func initConfig(config *Config) *Config {
	if config == nil {
		config = &Config{}
	}

	return config
}

// New creates a new health checker.
func New(config *Config) *HealthChecker {
	config = initConfig(config)

	checker := &HealthChecker{
		onlyDeadLinks: config.OnlyDeadLinks,
	}

	return checker
}

// Perform a deep search for all links inside the link and return a slice containing the result
func (checker *HealthChecker) GetLinks(link string) []LinkResult {
	parsedUrl, err := url.Parse(link)
	if err != nil {
		log.Fatal(err)
	}

	parsedStartUrl = parsedUrl

	g := &sync.WaitGroup{}
	g.Add(1)

	var links []LinkResult
	go checker.getLinksAux(link, &links, g)

	g.Wait()

	return links
}

func (checker *HealthChecker) getLinksAux(link string, result *[]LinkResult, g *sync.WaitGroup) {
	defer g.Done()
	resp, err := http.Get(link)

	if err != nil {
		log.Fatal(err)
	}

	body := resp.Body

	defer body.Close()

	tokenizer := html.NewTokenizer(body)

	for {
		tokenType := tokenizer.Next()

		switch {
		case tokenType == html.StartTagToken:

			token := tokenizer.Token()

			isLink := token.Data == "a"
			if !isLink {
				continue
			}

			href := getHref(token)
			href = removeSlash(href)

			if href != "" && indexOf(result, href) == -1 {
				parsedUrl, err := url.Parse(href)
				if err != nil {
					log.Printf("Invalid URL: %s", href)
					continue
				}

				if strings.HasPrefix(href, "#") || href == "//:0" {
					log.Printf("Invalid URL: %s", href)
					continue
				}

				if isSameDomain(parsedUrl) {
					fullUrl := getFullLink(parsedUrl)
					checker.addLink(result, LinkResult{Link: fullUrl, rawLink: href, StatusCode: resp.StatusCode})

					g.Add(1)
					go checker.getLinksAux(fullUrl, result, g)
				} else {
					checker.addLink(result, LinkResult{Link: href, rawLink: href, StatusCode: resp.StatusCode})
				}
			}

		case tokenType == html.ErrorToken:
			return
		}
	}
}

func getHref(token html.Token) string {
	for _, attr := range token.Attr {
		if attr.Key == "href" {
			return strings.TrimSpace(attr.Val)
		}
	}

	return ""
}

func indexOf(links *[]LinkResult, link string) int {
	for i, linkResult := range *links {
		if linkResult.rawLink == link {
			return i
		}
	}

	return -1
}

func isSameDomain(href *url.URL) bool {
	if strings.HasPrefix(href.String(), "/") {
		return true
	}

	if href.Host == parsedStartUrl.Host {
		return true
	}

	return false
}

func getFullLink(link *url.URL) string {
	if strings.HasPrefix(link.String(), "http") {
		return link.String()
	}

	return parsedStartUrl.ResolveReference(link).String()
}

func removeSlash(link string) string {
	if link == "/" {
		return link
	}

	return strings.TrimSuffix(link, "/")
}

func (checker *HealthChecker) addLink(linkResults *[]LinkResult, link LinkResult) {
	mutex.Lock()
	defer mutex.Unlock()

	if (checker.onlyDeadLinks && link.StatusCode != 200) || (!checker.onlyDeadLinks) {
		*linkResults = append(*linkResults, link)
	}
}
