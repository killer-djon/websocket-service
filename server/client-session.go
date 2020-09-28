package server

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
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
	HashKey string
	UserId int
	Key    string
	Room   string
	Peer   *websocket.Conn
}

type Peers struct {
	clients map[string][]*ClientSession
}

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
	for {
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
	}
}

func (p *Peers) GetClientChannels(key string) []*ClientSession {
	if client := p.clients[key]; client != nil {
		return client
	}
	return nil
}

/*func (p *Peers) Add(client ClientSession) *ClientSession {
	roomKey := MakeKeyHash(client.Key, client.UserId)
	if p.clients[roomKey] == nil {
		p.clients[roomKey] = &client
		p.clients[roomKey].HashKey = roomKey
		log.Println("New client connected", client)

		return p.clients[roomKey]
	}
	return nil
}

func (p *Peers) Del(key string) {
	if len(p.clients) > 0 {
		for _, client := range p.clients {
			if client.Key == key {
				log.Println("Remove client from slice", client)
				p.clients[client.HashKey].Peer.Close()
				delete(p.clients, client.HashKey)
			}
		}
	}
}

func (p *Peers) Get(key string) *ClientSession {
	if len(p.clients) > 0 {
		for _, client := range p.clients {
			if client.Key == key {
				log.Println("Finded client by room key", p.clients[client.HashKey])
				return p.clients[client.HashKey]
			}
		}
	}
	return nil
}


func (p *Peers) Start(client *ClientSession) {
	for {
		if _, _, err := p.clients[client.HashKey].Peer.NextReader(); err != nil {
			log.Println("Close connection for socket", p.clients[client.HashKey])
			p.Del(client.Key)
			break
		}
	}
}

func (p *Peers) List() map[string]*ClientSession {
	return p.clients
}*/


func MakeKeyHash(key string, id int) string {
	byteKeyRoom := []byte(fmt.Sprintf("%s_%d_%d", key, id, time.Now().UnixNano()))
	hashKey := sha1.New()
	hashKey.Write(byteKeyRoom)
	sha := base64.URLEncoding.EncodeToString(hashKey.Sum(nil))

	return sha
}