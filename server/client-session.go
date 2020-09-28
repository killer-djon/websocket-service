package server

import (
	"github.com/gorilla/websocket"
	"log"
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
	UserId int
	Key    string
	Room   string
	Peer   *websocket.Conn
}

type Peers struct {
	clients map[string]*ClientSession
}

func NewPeersConnection() *Peers {
	return &Peers{
		clients: make(map[string]*ClientSession),
	}
}

func (p *Peers) Add(client ClientSession) {
	if p.clients[client.Key] == nil {
		p.clients[client.Key] = &client
		log.Println("New client connected", client)
	}
}

func (p *Peers) Del(client ClientSession) {
	if p.clients[client.Key] != nil {
		log.Println("Remove client from slice", client)
		p.clients[client.Key].Peer.Close()
		delete(p.clients, client.Key)
	}
}


func (p *Peers) Start(key string) {
	for {
		if _, _, err := p.clients[key].Peer.NextReader(); err != nil {
			log.Println("Close connection for socket", p.clients[key])
			p.clients[key].Peer.Close()
			break
		}
	}
}

func (p *Peers) List() map[string]*ClientSession {
	return p.clients
}

func (p *Peers) Get(key string) *ClientSession{
	return p.clients[key]
}