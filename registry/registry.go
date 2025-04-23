package registry

// 注册中心仅负责维护活跃的服务器，并不负责维护服务器具体提供哪些方法
// GeeRegistry is a simple register center, provide following functions.
// add a server and receive heartbeat to keep it alive.
// returns all alive servers and delete dead servers sync simultaneously.

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type ServerItem struct {
	Addr  string
	start time.Time
}

type GeeRegistry struct {
	timeout time.Duration
	mu      sync.Mutex
	servers map[string]*ServerItem
}

const (
	defaultPath    = "/_geerpc_/registry"
	defaultTimeout = time.Minute * 5
)

var defaultGeeRegistry = NewGeeRegistry(defaultTimeout)

func NewGeeRegistry(timeout time.Duration) *GeeRegistry {
	return &GeeRegistry{
		servers: make(map[string]*ServerItem),
		timeout: timeout,
	}
}

func (r *GeeRegistry) putServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s := r.servers[addr]
	if s == nil {
		r.servers[addr] = &ServerItem{
			Addr:  addr,
			start: time.Now(),
		}
	} else {
		s.start = time.Now()
	}
}

func (r *GeeRegistry) aliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var alives []string
	for addr, s := range r.servers {
		if r.timeout == 0 || s.start.Add(r.timeout).After(time.Now()) {
			alives = append(alives, addr)
		} else {
			delete(r.servers, addr)
		}
	}
	sort.Strings(alives)
	return alives
}

func (r *GeeRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		// keep it simple, server is in req.Header
		w.Header().Set("X-Geerpc-Servers", strings.Join(r.aliveServers(), ","))
	case "POST":
		// keep it simple, server is in req.Header
		addr := req.Header.Get("X-Geerpc-Server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *GeeRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r)
	log.Println("rpc registry path:", registryPath)
}

func HandleHTTP() {
	defaultGeeRegistry.HandleHTTP(defaultPath)
}

// Heartbeat send a heartbeat message every once in a while
// it's a helper function for a server to register or send heartbeat
func Heartbeat(registry, addr string, duration time.Duration) {
	if duration == 0 {
		// make sure there is enough time to send heart beat
		// before it's removed from registry
		// 超时时间 减 1分钟
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	err = sendHeartbeat(registry, addr)
	go func() {
		t := time.NewTicker(duration)
		for err == nil {
			<-t.C
			err = sendHeartbeat(registry, addr)
		}
	}()
}

func sendHeartbeat(registry, addr string) error {
	log.Println(addr, "send heart beat to registry", registry)
	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("X-Geerpc-Server", addr)
	if _, err := httpClient.Do(req); err != nil {
		log.Println("rpc server: heart beat err:", err)
		return err
	}
	return nil
}
