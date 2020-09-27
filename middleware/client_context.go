package middleware

import (
	"github.com/gorilla/websocket"
)

type Client struct {
	Room    string
	RoomKey string
	Id      int
	WsConn *websocket.Conn
}