package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	JSON_FILE = "./config.json"
	LOG_FILE  = "/var/log/websocket-service/websocket-service.log"
)

type ServerConfig struct {
	Host              string `json:"host"`
	Port              int    `json:"port"`
	EndPoint          string `json:"end_point"`
	UseSsl            bool   `json:"use_ssl"`
	SslCertificate    string `json:"ssl_certificate"`
	SslCertificateKey string `json:"ssl_certificate_key"`
}

type Config struct {
	Server ServerConfig
}

type WsClient struct {
	Room       string
	RoomKey    string
	Id         int
	ClientConn []*websocket.Conn
}

var WsClientList = make(map[string]*WsClient)

var (
	configFile string
	logFile    string
	upgrader   = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func init() {
	flag.StringVar(&configFile, "configFile", JSON_FILE, "Type your config file for parse them")
	flag.StringVar(&logFile, "logFile", LOG_FILE, "Set log file for debug info")
	flag.Parse()

	if _, err := os.Stat("/var/log/websocket-service"); os.IsNotExist(err) {
		os.MkdirAll("/var/log/websocket-service", 0775)
	}
	wrt := io.MultiWriter(os.Stdout)
	log.SetOutput(wrt)
}

func parseJson(jsonConfig string) *Config {
	jsonFile, err := os.Open(jsonConfig)

	// if we os.Open returns an error then handle it
	if err != nil {
		log.Printf("Error when try to open json config file %v", err)
		return nil
	}

	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)

	var rootConfig Config
	err = json.Unmarshal(byteValue, &rootConfig)

	if err != nil {
		log.Printf("Cant read json root data %v", err)
		return nil
	}

	return &rootConfig
}

func main() {
	config := parseJson(configFile)
	if config == nil {
		log.Println("Error to read config struct from json file")
		return
	}

	serverConfig := config.Server

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// Routes list
	//r.Get("/ws/{key}-{id}", listenPaymentWaitSocket)
	//r.Post("/publish", publishPaymentWaitChannel)

	r.Get("/room/{room}/{key}/{id}", startListenSocket)
	r.Post("/publish/{room}/{key}/{id}", startPublishToSocket)

	// Start listen server by config
	log.Println("Start listen server at ", fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port))

	var err error
	if serverConfig.UseSsl == true {
		err = http.ListenAndServeTLS(
			fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port),
			serverConfig.SslCertificate,
			serverConfig.SslCertificateKey,
			r)
	} else {
		err = http.ListenAndServe(fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port), r)
	}

	if err != nil {
		log.Println("Error to start listen server", err)
	}
}

//
//func listenPaymentWaitSocket(writer http.ResponseWriter, request *http.Request) {
//	peer, err := upgrader.Upgrade(writer, request, nil)
//	if err != nil {
//		log.Fatal("websocket conn failed", err)
//	}
//
//	key := chi.URLParam(request, "key")
//	id, _ := strconv.Atoi(chi.URLParam(request, "id"))
//
//
//}
//
//func publishPaymentWaitChannel(writer http.ResponseWriter, request *http.Request) {
//	query := request.URL.Query()
//	if query["id"] == nil {
//		log.Println("Cant publish to socket channel")
//		return
//	}
//
//	if query["id"] != nil {
//		strSplited := strings.Split(query["id"][0], "-")
//		key := strSplited[0]
//		id, _ := strconv.Atoi(strSplited[1])
//
//		if clients := peers.GetClientChannels(fmt.Sprintf("%s_%d", key, id)); clients != nil {
//			body, err := ioutil.ReadAll(request.Body)
//			if err != nil {
//				log.Println("Error to read body request", err)
//				return
//			}
//
//			defer request.Body.Close()
//			for _, client := range clients {
//				client.Peer.WriteMessage(websocket.TextMessage, body)
//			}
//		}
//	}
//}

func startListenSocket(writer http.ResponseWriter, request *http.Request) {
	ws, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Fatal("websocket conn failed", err)
	}

	room := chi.URLParam(request, "room")
	key := chi.URLParam(request, "key")
	id, _ := strconv.Atoi(chi.URLParam(request, "id"))

	clientWsChannel := fmt.Sprintf("%s_%s_%d", room, key, id)
	if WsClientList[clientWsChannel] == nil {
		var connPool = make([]*websocket.Conn, 0)
		connPool = append(connPool, ws)
		WsClientList[clientWsChannel] = &WsClient{
			Room:       room,
			RoomKey:    key,
			Id:         id,
			ClientConn: connPool,
		}
	} else {
		WsClientList[clientWsChannel].ClientConn = append(
			WsClientList[clientWsChannel].ClientConn,
			ws,
		)
	}

	go writePump(WsClientList[clientWsChannel])
	log.Println("Connect clients instance", WsClientList[clientWsChannel].ClientConn)
}

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

func writePump(client *WsClient) {
	c := time.Tick(pingPeriod)
	go func(){
		select {
			case <- c:
				log.Println("Try to send PING message control to client", client.ClientConn)
				if client.ClientConn != nil {
					for _, cl := range client.ClientConn {
						if err := cl.WriteControl(websocket.PingMessage, []byte("\n"), time.Now().Add(writeWait)); err != nil {
							log.Println("Handle error for write control message ping", err)
						}
					}
				}
		}
	}()

	for key, cl := range client.ClientConn {
		cl.SetPongHandler(func(string) error { return cl.SetReadDeadline(time.Now().Add(pingPeriod)) })
		go func(key int){
			for {
				log.Println("Try to send PONG message control to client", client.Id, client.RoomKey, cl)
				if cl == nil {
					log.Println("Client disconnected", client)
					break
				}else{
					if _, _, err := cl.NextReader(); err != nil {
						log.Println("Close connection for socket", client)
						err := cl.Close()
						if err !=nil {
							log.Println("Error for close connection", cl)
						}
						break
					}
				}
			}
		}(key)
	}
}

type BodyResponse struct {
	Body   []byte
	Client *WsClient
}
var bodyResponse = make(chan BodyResponse)
func startPublishToSocket(writer http.ResponseWriter, request *http.Request) {
	room := chi.URLParam(request, "room")
	key := chi.URLParam(request, "key")
	id, _ := strconv.Atoi(chi.URLParam(request, "id"))

	log.Println("Select from all clients for published", WsClientList)
	clientWsChannel := fmt.Sprintf("%s_%s_%d", room, key, id)

	if WsClientList[clientWsChannel] != nil {

		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			log.Println("Error to read body request", err)
			return
		}
		defer request.Body.Close()


		go func() {
			for {
				bodyResponse <- BodyResponse{
					Body:   body,
					Client: WsClientList[clientWsChannel],
				}
			}
		}()

		writeToChannel(bodyResponse)
	}

}

func writeToChannel(body <-chan BodyResponse) {
	client := <-body
	log.Println("Published for client", client.Body, client.Client)
	for _, cl := range client.Client.ClientConn {
		if cl != nil {
			cl.WriteMessage(websocket.TextMessage, client.Body)
		}
	}
}
