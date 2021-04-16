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

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Pool maintains the set of active clients and broadcasts messages to the
// clients.
type Pool struct {
	// Registered clients.
	Clients map[string]map[*Client]bool

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	// Channel websocket key
	ChannelKey string

	// Clients pool connections
	Pool *Pool

	// The websocket connection.
	Conn *websocket.Conn

	Send chan []byte
}

// NewClientPool make new pool clients
func NewClientPool() *Pool {
	return &Pool{
		Clients:    make(map[string]map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

// StartCollector start collection for clients
func (pool *Pool) StartCollector() {
	var clientMap = make(map[*Client]bool)
	for {
		select {
		case client := <-pool.Register:
			clientMap[client] = true
			pool.Clients[client.ChannelKey] = clientMap
		case client := <-pool.Unregister:
			if clKey, ok := pool.Clients[client.ChannelKey]; ok {
				if active := clKey[client]; active {
					log.Println("Before: Unregister client", client, pool.Clients[client.ChannelKey])
					client.Conn.Close()
					delete(pool.Clients[client.ChannelKey], client)
					log.Println("After: Unregister client", client, pool.Clients[client.ChannelKey])
				}
			}
		}
	}
}

func (c *Client) ReadPump() {
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			c.Pool.Unregister <- c
			break
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	stopped := make(chan bool, 1)
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				log.Println("Close channel message")
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.Pool.Unregister <- c
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Println("Error for next writer", err)
				return
			}

			byteWrite, err := w.Write(message)
			if err != nil {
				log.Println("Error write message to channel", c.ChannelKey, err)
				return
			}

			log.Println("Write message to channel by byte count", c.ChannelKey, byteWrite)

			if err := w.Close(); err != nil {
				log.Println("Close channel for client", c.ChannelKey)
				return
			}
		case <-stopped:
			ticker.Stop()
			return
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			log.Println("Write ping message to client", c.ChannelKey)
			if err := c.Conn.WriteControl(websocket.PingMessage, newline, time.Now().Add(writeWait)); err != nil {
				log.Println("Client channel is closed", c.ChannelKey)
				c.Pool.Unregister <- c
				return
			}
		}
	}
}
