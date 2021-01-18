package main

import (
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/burnjake/loadbalancer/internal/metrics"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	httpPort                  string = ":8090"
	tcpPort                   string = ":5353"
	metricsPort               string = ":8091"
	bufferSize                int    = 4096
	healthCheckCadenceSeconds int    = 5
	healthCheckTimeoutSeconds int    = 1
)

// Target defines a destination server
type Target struct {
	address string
	healthy bool
	mu      sync.RWMutex
}

// Pool stores the list of targets and the global indexer
type Pool struct {
	pool          []*Target
	atomicCounter uint64
}

func (target *Target) checkHealth() {
	tcpConn, err := net.DialTimeout(
		"tcp4",
		target.address,
		time.Duration(healthCheckTimeoutSeconds)*time.Second,
	)

	if err != nil {
		target.mu.Lock()
		target.healthy = false
		target.mu.Unlock()
	} else {
		target.mu.Lock()
		target.healthy = true
		target.mu.Unlock()
		defer tcpConn.Close()
	}
}

func (pool *Pool) getNextTarget() (*Target, error) {
	for i := 0; i < len(pool.pool); i++ {
		index := int(atomic.AddUint64(&pool.atomicCounter, 1))
		potentialTarget := pool.pool[index%len(pool.pool)]
		potentialTarget.mu.Lock()
		defer potentialTarget.mu.Unlock()
		if potentialTarget.healthy == true {
			return potentialTarget, nil
		}
	}

	return nil, errors.New("pool has no healthy targets")
}

func (pool *Pool) loadBalanceHTTP(w http.ResponseWriter, req *http.Request) {
	target, err := pool.getNextTarget()
	if err != nil {
		log.Printf("Error fetching next target: %s", err)
		return
	}
	url, _ := url.Parse("http://" + target.address)
	proxy := httputil.NewSingleHostReverseProxy(url)
	log.Printf("Loadbalancing to %s", url)
	proxy.ServeHTTP(w, req)
}

func (pool *Pool) loadBalanceTCP(clientConn net.Conn) {
	// Deferring here as the caller function will never return (infinite while loop)
	defer clientConn.Close()

	target, err := pool.getNextTarget()
	if err != nil {
		log.Printf("Error fetching next target: %s", err)
		return
	}

	remoteConn, err := net.Dial("tcp4", target.address)
	if err != nil {
		log.Printf("Error establishing tcp connection to %s. %s", target.address, err)
		return
	}
	defer remoteConn.Close()

	log.Printf("Loadbalancing to %s", remoteConn.RemoteAddr())

	// request
	proxyConn(clientConn, remoteConn)
	// response
	proxyConn(remoteConn, clientConn)
}

func proxyConn(source, dest net.Conn) {
	var buffer [bufferSize]byte
	n, err := source.Read(buffer[0:])
	if err != nil {
		log.Printf("Error reading from source connection: %s", err)
		return
	}
	n, err = dest.Write(buffer[:n])
	if err != nil {
		log.Printf("Error writing to dest connection: %s", err)
		return
	}
}

func main() {
	proto := flag.String("protocol", "http", "Valid options are tcp and http.")
	flag.Parse()

	var pool Pool

	pool.pool = []*Target{
		{address: "tcpbin.com:4242"},
		{address: "tcpbin.com:4343"},
		{address: "tcpbin.com:4444"},
	}

	metrics.NumTargets.Set(float64(len(pool.pool)))

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(metricsPort, nil)
	}()

	go func(targets []*Target) {
		for {
			healthyTargets := 0
			for _, target := range targets {
				target.checkHealth()
				if target.healthy == true {
					healthyTargets++
				}
			}
			metrics.NumHealthyTargets.Set(float64(healthyTargets))
			time.Sleep(time.Duration(healthCheckCadenceSeconds) * time.Second)
		}
	}(pool.pool)

	switch *proto {
	case "http":
		log.Println("Loadbalancing via http...")
		http.HandleFunc("/", pool.loadBalanceHTTP)
		err := (http.ListenAndServe(httpPort, nil))
		if err != nil {
			log.Println(err)
		}
	case "tcp":
		log.Println("Loadbalancing via tcp...")
		listener, err := net.Listen("tcp4", tcpPort)
		if err != nil {
			log.Println(err)
		}
		defer listener.Close()
		for {
			// Wait for connection
			conn, err := listener.Accept()
			if err != nil {
				log.Println(err)
				return
			}
			metrics.TCPConnectionsCounter.Add(1)
			go pool.loadBalanceTCP(conn)
		}
	default:
		log.Fatal("Unsupported protocol:", *proto)
	}
}
