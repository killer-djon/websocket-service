package server

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"sync"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type ClientSession struct {
	HashKey string
	UserId  int
	Key     string
	Room    *string
	Peer    *websocket.Conn
	mu      sync.Mutex
}

type Peers struct {
	clients map[string][]*ClientSession
}

// NewPeersConnection make peer slice clients
// for collect all connected clients to socket channels
func NewPeersConnection() *Peers {
	return &Peers{
		clients: make(map[string][]*ClientSession),
	}
}

func (p *Peers) AddClient(session *ClientSession) *ClientSession {
	sessionKey := fmt.Sprintf("%s_%d", session.Key, session.UserId)

	cl := session
	cl.HashKey = sessionKey
	p.clients[sessionKey] = append(p.clients[sessionKey], cl)

	log.Println("Client was added", cl)
	return cl
}

func (p *Peers) RemoveClient(key string, index int) []*ClientSession {
	return append(p.clients[key][:index], p.clients[key][index+1:]...)
}

func (p *Peers) Start(client *ClientSession) {
	c := time.Tick(pingPeriod)
	go func() {
		for range c {
			log.Println("Try to send PING message control to client")
			if client.Peer != nil {
				sessionKey := fmt.Sprintf("%s_%d", client.Key, client.UserId)
				log.Println("All client session", p.clients[sessionKey])
				for _, cl := range p.clients[sessionKey] {
					if err := cl.Peer.WriteControl(websocket.PingMessage, []byte("\n"), time.Now().Add(writeWait)); err != nil {
						log.Println("Handle error for write control message ping", err)
					}
				}
			}
		}
	}()

	client.Peer.SetPongHandler(func(string) error { return client.Peer.SetReadDeadline(time.Now().Add(pingPeriod)) })
	go func() {
		for {
			log.Println("Try to send PONG message control to client")
			if client.Peer == nil {
				log.Println("Client disconnected", client)
				break
			} else {
				if _, _, err := client.Peer.NextReader(); err != nil {
					log.Println("Close connection for socket", client)
					sessionKey := fmt.Sprintf("%s_%d", client.Key, client.UserId)
					for i, cl := range p.clients[sessionKey] {
						if cl.Peer == client.Peer {
							p.clients[sessionKey] = p.RemoveClient(sessionKey, i)
							cl.Peer.Close()

						}
					}
					break
				}
			}
		}
	}()

	/*for {
		if _, _, err := client.Peer.NextReader(); err != nil {
			log.Println("Close connection for socket", client)
			sessionKey := fmt.Sprintf("%s_%d", client.Key, client.UserId)
			for i, cl := range p.clients[sessionKey] {
				if cl.Peer == client.Peer {
					p.clients[sessionKey] = p.RemoveClient(sessionKey, i)
				}
			}
			break
		}
	}*/
}

func (p *Peers) GetClientChannels(key string) []*ClientSession {
	if client := p.clients[key]; client != nil {
		return client
	}
	return nil
}

func MakeKeyHash(key string, id int) string {
	byteKeyRoom := []byte(fmt.Sprintf("%s_%d_%d", key, id, time.Now().UnixNano()))
	hashKey := sha1.New()
	hashKey.Write(byteKeyRoom)
	sha := base64.URLEncoding.EncodeToString(hashKey.Sum(nil))

	return sha
}

func (client *ClientSession) Publish(body []byte) {
	client.mu.Lock()
	defer client.mu.Unlock()

	client.Peer.WriteMessage(websocket.TextMessage, body)
}
