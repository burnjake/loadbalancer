package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

const (
	httpPort string = ":80"
	tcpPort  string = ":7006"
)

// Targets stores the pool of URLs and the global indexer
type Targets struct {
	targets       []string
	atomicCounter uint64
}

func getNextTarget(targets *Targets) string {
	atomic.AddUint64(&targets.atomicCounter, 1)
	return targets.targets[int(targets.atomicCounter)%len(targets.targets)]
}

func proxyConn(source, dest net.Conn) (net.Conn, error) {
	clientRequest, err := ioutil.ReadAll(source)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	log.Printf("Source message from %s: %s\n", source.LocalAddr(), string(clientRequest))

	_, err = dest.Write(clientRequest)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return dest, nil
}

func (targets *Targets) loadBalanceHTTP(w http.ResponseWriter, req *http.Request) {
	url, _ := url.Parse(getNextTarget(targets))
	proxy := httputil.NewSingleHostReverseProxy(url)
	log.Printf("Loadbalancing to %s", url)
	proxy.ServeHTTP(w, req)
}

func (targets *Targets) loadBalanceTCP(clientConn net.Conn) {
	// Deferring here as the caller function will never return (infinite while loop)
	// defer clientConn.Close()

	target := getNextTarget(targets)

	remoteConn, err := net.Dial("tcp4", target)
	if err != nil {
		log.Println(err)
		return
	}
	defer remoteConn.Close()

	log.Printf("Loadbalancing to %s\n", remoteConn.RemoteAddr())

	remoteConn, err = proxyConn(clientConn, remoteConn)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = proxyConn(remoteConn, clientConn)
	if err != nil {
		log.Println(err)
		return
	}
}

func main() {
	proto := flag.String("protocol", "http", "Valid options are tcp and http. Defaults to http.")
	flag.Parse()

	targets := Targets{
		// targets: []string{"www.bbc.com:80", "www.google.com:80", "www.amazon.com:80"},
		targets: []string{"tcpbin.com:4242"},
	}

	switch *proto {
	case "http":
		log.Println("Loadbalancing via http...")
		http.HandleFunc("/", targets.loadBalanceHTTP)
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
			go targets.loadBalanceTCP(conn)
		}
	default:
		log.Fatal("Unsupported protocol:", *proto)
	}
}
