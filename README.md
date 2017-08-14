momus
========

[![GoDoc](https://godoc.org/github.com/fagnercarvalho/momus?status.svg)](https://godoc.org/github.com/fagnercarvalho/momus)

momus is a web scraper written in Go made to health check all the internal links inside a given site.

## Quick Start

```go
package main

import (
	"fmt"
	"github.com/fagnercarvalho/momus"
)

func main() {
	healthChecker := momus.New(&momus.Config{OnlyDeadLinks: false})
	links := healthChecker.GetLinks("http://fagner.co")

	for _, linkResult := range links {
		fmt.Printf("%d | %s \n", linkResult.StatusCode, linkResult.Link)
	}
}
```

## Example

http://www.github.com/fagnercarvalho/momus-example