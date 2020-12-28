package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

// Targets stores the pool of URLs and the global indexer
type Targets struct {
	targets []string
	counter uint64
}

func getNextTarget(targets *Targets) string {
	atomic.AddUint64(&targets.counter, 1)
	return targets.targets[int(targets.counter)%len(targets.targets)]
}

func (targets *Targets) loadBalance(w http.ResponseWriter, req *http.Request) {
	url, _ := url.Parse(getNextTarget(targets))
	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.ServeHTTP(w, req)
}

func main() {
	targets := Targets{
		targets: []string{"https://www.bbc.com", "https://www.google.com", "https://www.amazon.com"},
	}

	http.HandleFunc("/", targets.loadBalance)
	log.Fatal(http.ListenAndServe(":80", nil))
}
