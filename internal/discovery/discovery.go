package discovery

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Peer struct {
	Username string
	IP       string
	Port     string
	FullName string
	LastSeen time.Time
}

type Node struct {
	Username string
	FullName string
	Port     string
	Peers    map[string]Peer
	mu       sync.Mutex
	quit     chan struct{}
}

func NewNode(username, fullName, port string) *Node {
	return &Node{
		Username: username,
		FullName: fullName,
		Port:     port,
		Peers:    make(map[string]Peer),
		quit:     make(chan struct{}),
	}
}

func (n *Node) Start() error {
	return n.startMulticast()
}

func (n *Node) startMulticast() error {
	addr, err := net.ResolveUDPAddr("udp", "239.255.255.250:9999")
	if err != nil {
		return err
	}
	
	// Start Broadcasting
	go func() {
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			return
		}
		defer conn.Close()
		
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-n.quit:
				return
			case <-ticker.C:
				msg, _ := json.Marshal(map[string]string{
					"username": n.Username,
					"fullname": n.FullName,
					"port":     n.Port,
				})
				conn.Write(msg)
			}
		}
	}()

	// Start Listening
	go func() {
		ifi, _ := net.Interfaces()
		for _, i := range ifi {
			if i.Flags&net.FlagMulticast != 0 && i.Flags&net.FlagUp != 0 {
				listenConn, err := net.ListenMulticastUDP("udp", &i, addr)
				if err == nil {
					go n.listenMulticast(listenConn)
				}
			}
		}
	}()

	return nil
}

func (n *Node) listenMulticast(conn *net.UDPConn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	for {
		select {
		case <-n.quit:
			return
		default:
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			nLen, remoteAddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				continue
			}
			
			var pInfo map[string]string
			if err := json.Unmarshal(buf[:nLen], &pInfo); err == nil {
				if pInfo["username"] != n.Username && pInfo["username"] != "" {
					ip := remoteAddr.IP.String()
					// Ignore localhost packets if any
					if !strings.HasPrefix(ip, "127.") && ip != "::1" {
						n.mu.Lock()
						n.Peers[pInfo["username"]] = Peer{
							Username: pInfo["username"],
							FullName: pInfo["fullname"],
							IP:       ip,
							Port:     pInfo["port"],
							LastSeen: time.Now(),
						}
						n.mu.Unlock()
					}
				}
			}
		}
	}
}

func (n *Node) Stop() {
	close(n.quit)
}

func (n *Node) GetPeers() []Peer {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	var active []Peer
	now := time.Now()
	for k, p := range n.Peers {
		if now.Sub(p.LastSeen) < 10*time.Second {
			active = append(active, p)
		} else {
			delete(n.Peers, k)
		}
	}
	return active
}

func (n *Node) StartServer(onMessage func([]byte)) error {
	http.HandleFunc("/p2p", func(w http.ResponseWriter, r *http.Request) {
		var body []byte
		if r.Body != nil {
			buf := new(strings.Builder)
			_, _ = io.Copy(buf, r.Body)
			body = []byte(buf.String())
			if onMessage != nil {
				onMessage(body)
			}
		}
		w.WriteHeader(http.StatusOK)
	})
	return http.ListenAndServe(":"+n.Port, nil)
}

func (n *Node) SendMessage(ip, port string, data []byte) error {
	url := "http://" + ip + ":" + port + "/p2p"
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
