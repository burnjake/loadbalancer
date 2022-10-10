package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/burnjake/loadbalancer/internal/metrics"
	"gopkg.in/yaml.v2"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const bufferSize int = 4096

var conf Config

// Config defines the app config structure as read in from a yaml file
type Config struct {
	Addresses string `yaml:"addresses"`
	Protocol  string `yaml:"protocol"`
	Ports     struct {
		TCP     string `yaml:"tcp"`
		HTTP    string `yaml:"http"`
		Metrics string `yaml:"metrics"`
	} `yaml:"ports"`
	HealthCheck struct {
		CadenceSeconds int `yaml:"cadenceSeconds"`
		TimeoutSeconds int `yaml:"timeoutSeconds"`
	} `yaml:"healthCheck"`
}

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

func (config *Config) readConfig(filepath string) {
	contents, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalf("Error reading from config file: %s", err)
	}

	err = yaml.Unmarshal(contents, config)
	if err != nil {
		log.Fatalf("Error parsing yaml: %s", err)
	}
}

func (target *Target) checkHealth() {
	tcpConn, err := net.DialTimeout(
		"tcp4",
		target.address,
		time.Duration(conf.HealthCheck.TimeoutSeconds)*time.Second,
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
		if potentialTarget.healthy {
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
		metrics.TCPConnectionsCounter.Add(1)
		return
	}
	defer remoteConn.Close()

	log.Printf("Loadbalancing to %s", target.address)

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
	_, err = dest.Write(buffer[:n])
	if err != nil {
		log.Printf("Error writing to dest connection: %s", err)
		return
	}
}

func makeTargets(addresses string) []*Target {
	var targets []*Target
	for _, address := range strings.Split(addresses, ",") {
		targets = append(targets, &Target{address: address})
		log.Printf("Parsed target: %s", address)
	}
	return targets
}

func main() {
	filepath := flag.String("config", "/opt/loadbalancer/config.yaml", "Config file path")
	flag.Parse()

	var pool Pool

	conf.readConfig(*filepath)

	pool.pool = makeTargets(conf.Addresses)

	metrics.NumTargets.Set(float64(len(pool.pool)))

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":"+conf.Ports.Metrics, nil)
	}()

	go func(targets []*Target) {
		for {
			healthyTargets := 0
			for _, target := range targets {
				target.checkHealth()
				if target.healthy {
					healthyTargets++
				}
			}
			metrics.NumHealthyTargets.Set(float64(healthyTargets))
			time.Sleep(time.Duration(conf.HealthCheck.CadenceSeconds) * time.Second)
		}
	}(pool.pool)

	switch conf.Protocol {
	case "http":
		log.Println("Loadbalancing via http...")
		http.HandleFunc("/", pool.loadBalanceHTTP)
		err := (http.ListenAndServe(":"+conf.Ports.HTTP, nil))
		if err != nil {
			log.Println(err)
		}
	case "tcp":
		log.Println("Loadbalancing via tcp...")
		listener, err := net.Listen("tcp4", ":"+conf.Ports.TCP)
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
			go pool.loadBalanceTCP(conn)
		}
	default:
		log.Fatal("Unsupported protocol:", conf.Protocol)
	}
}
