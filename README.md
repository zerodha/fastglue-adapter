# Fastglue-Adapter [![Go Reference](https://pkg.go.dev/badge/github.com/zerodha/fastglue-adapter.svg)](https://pkg.go.dev/github.com/zerodha/fastglue-adapter) [![Zerodha Tech](https://zerodha.tech/static/images/github-badge.svg)](https://zerodha.tech)
Helper functions for converting net/http request handlers to fastglue request handlers.

## Overview
A port from [fasthttpadaptor](https://github.com/valyala/fasthttp/tree/master/fasthttpadaptor). While this function may be used for easy switching from net/http to fastglue, it has the following drawbacks comparing to using manually written fastglue
 request handler:

* A lot of useful functionality provided by fastglue is missing from net/http handler such as webhooks.
* net/http -> fastglue handler conversion has some overhead,so the returned handler will be always slower than manually written fastglue handler.

So it is advisable using this function only for quick `net/http` -> `fastglue` switching or unavoidable situations. 

## Install

```bash
go get -u github.com/zerodha/fastglue-adapter
```

## Usage

```go
import "github.com/zerodha/fastglue-adapter"
```

## Example

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	fga "github.com/zerodha/fastglue-adapter"
)

func request(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %q", r.URL.Path)
}
func main() {
	g := fastglue.NewGlue()
	g.GET("/", fga.NewFastGlueHandlerFunc(request))

	s := &fasthttp.Server{}
	if err := g.ListenAndServe(":8000", "", s); err != nil {
		log.Fatal(err.Error())
	}
}
```
## [License](https://github.com/zerodha/fastglue-adapter/blob/master/LICENSE)
The MIT License (MIT)

Copyright (c) 2020 Zerodha Technology Pvt. Ltd. (India)
