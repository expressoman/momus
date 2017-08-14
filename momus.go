// Package momus is a web scraper made to health check all the internal links inside a given site.
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
	indexOfMutex   sync.Mutex
	addMutex       sync.Mutex
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

// Perform a deep search for all internal links inside the link and return a slice containing the result
func (checker *HealthChecker) GetLinks(link string) []LinkResult {
	parsedUrl, err := url.Parse(link)
	if err != nil {
		log.Fatal(err)
	}

	parsedStartUrl = parsedUrl

	g := &sync.WaitGroup{}
	g.Add(1)

	var links []LinkResult
	go checker.getLinksAux(parsedUrl, link, &links, g)

	g.Wait()

	return links
}

func (checker *HealthChecker) getLinksAux(link *url.URL, rawLink string, result *[]LinkResult, g *sync.WaitGroup) {
	defer g.Done()
	resp, err := http.Get(link.String())

	if err != nil {
		log.Println(err)
		checker.addLink(result, LinkResult{Link: link.String(), StatusCode: 0, rawLink: rawLink})
		return
	}

	checker.addLink(result, LinkResult{Link: link.String(), StatusCode: resp.StatusCode, rawLink: rawLink})

	if resp.StatusCode != http.StatusOK {
		return
	}

	body := resp.Body

	defer body.Close()

	if !isSameDomain(link) {
		return
	}

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

			if indexOf(result, href) != -1 {
				continue
			}

			parsedUrl, err := url.Parse(href)
			if err != nil {
				continue
			}

			if href == "" || strings.HasPrefix(href, "#") || href == "//:0" {
				continue
			}

			if !isSameDomain(parsedUrl) {
				continue
			}

			parsedFullUrl := getFullLink(parsedUrl)

			g.Add(1)

			go checker.getLinksAux(parsedFullUrl, href, result, g)

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
	indexOfMutex.Lock()
	defer indexOfMutex.Unlock()

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

func getFullLink(link *url.URL) *url.URL {
	if strings.HasPrefix(link.String(), "http") {
		return link
	}

	return parsedStartUrl.ResolveReference(link)
}

func removeSlash(link string) string {
	if link == "/" {
		return link
	}

	return strings.TrimSuffix(link, "/")
}

func (checker *HealthChecker) addLink(linkResults *[]LinkResult, link LinkResult) {
	addMutex.Lock()
	defer addMutex.Unlock()

	if indexOf(linkResults, link.rawLink) != -1 {
		return
	}

	if (checker.onlyDeadLinks && link.StatusCode != 200) || (!checker.onlyDeadLinks) {
		*linkResults = append(*linkResults, link)
	}
}
