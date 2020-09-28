package main

import (
	"bitbucket.org/projectt_ct/websocker-service/server"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
)

const (
	JSON_FILE = "./config.json"
)

type WsClient struct {
	Room    string
	RoomKey string
	Id      int
	WsConn  *websocket.Conn
}

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

type WsClientList map[string]*WsClient

var (
	configFile      string
	upgrader        = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	clientConnected = make(WsClientList)
)

var peers *server.Peers

func init() {
	flag.StringVar(&configFile, "configFile", JSON_FILE, "Type your config file for parse them")
	flag.Parse()
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
	peers = server.NewPeersConnection()
	serverConfig := config.Server

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	//r.Use(middleware.AllowContentType("application/json"))
	// Routes list
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
	}else{
		err = http.ListenAndServe(fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port), r)
	}

	if err != nil {
		log.Println("Error to start listen server", err)
	}
}

func startListenSocket(writer http.ResponseWriter, request *http.Request) {
	peer, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Fatal("websocket conn failed", err)
	}

	room := chi.URLParam(request, "room")
	key := chi.URLParam(request, "key")
	id, _ := strconv.Atoi(chi.URLParam(request, "id"))

	go func() {
		newClient := peers.AddClient(&server.ClientSession{
			UserId: id,
			Key:    key,
			Room:   room,
			Peer:   peer,
		})
		peers.Start(newClient)
	}()
}

func startPublishToSocket(writer http.ResponseWriter, request *http.Request) {
	key := chi.URLParam(request, "key")
	id, _ := strconv.Atoi(chi.URLParam(request, "id"))

	log.Println("Key for published", fmt.Sprintf("%s_%d", key, id))
	if clients := peers.GetClientChannels(fmt.Sprintf("%s_%d", key, id)); clients != nil {
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			log.Println("Error to read body request", err)
			return
		}
		defer request.Body.Close()
		for _, client := range clients {
			client.Peer.WriteMessage(websocket.TextMessage, body)
		}
	}
}
