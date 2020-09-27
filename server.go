package main

import (
	ml "bitbucket.org/projectt_ct/websocker-service/middleware"
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

var (
	configFile string
	upgrader   = websocket.Upgrader{}
)

type Config struct {
	Server ServerConfig
}

type ClientList map[int]*ml.Client

type ServerConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	EndPoint string `json:"end_point"`
}

var clientConnected = make(ClientList)

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

	serverConfig := config.Server
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json"))
	// Routes list
	r.Get(serverConfig.EndPoint, listClientSocket)
	r.Post("/publish/{clientRoom}/{userId}", publishMessageToClient)

	log.Println("Start listen server at ", fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port))
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port), r)
	if err != nil {
		log.Println("Error to start listen server", err)
	}
}

func publishMessageToClient(writer http.ResponseWriter, request *http.Request) {
	clientRoom := chi.URLParam(request, "clientRoom")
	userId, _ := strconv.Atoi(chi.URLParam(request, "userId"))

	log.Println("Must be publish to client room", clientRoom, userId)

	if clientRoom == "" || userId == 0 {
		log.Println("Published room is empty, roomKey is not set or userId is empty", clientRoom, userId)
	}

	var toPublishClient = clientConnected[userId]
	if toPublishClient == nil {
		log.Println("Client for published is not connected now", userId)
		return
	}

	bodyBytes, _ := ioutil.ReadAll(request.Body)
	//log.Println("Request body to publish", string(bodyBytes), toPublishClient)
	if len(bodyBytes) > 0 {
		var bodyToPublish interface{}
		json.Unmarshal(bodyBytes, &bodyToPublish)

		err := toPublishClient.WsConn.WriteJSON(bodyToPublish)
		if err != nil {
			log.Println("Error to publish message to user socket channel", err)
			return
		}
	}

}

func listClientSocket(writer http.ResponseWriter, request *http.Request) {
	wsConn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Println("Cant send upgrade status for client", err)
		return
	}

	defer wsConn.Close()

	room, roomKey := chi.URLParam(request, "room"), chi.URLParam(request, "key")
	userId, _ := strconv.Atoi(chi.URLParam(request, "id"))

	if clientConnected[userId] == nil {
		clientConnected[userId] = &ml.Client{
			Room:    room,
			RoomKey: roomKey,
			Id:      userId,
			WsConn:  wsConn,
		}
	}

	log.Println("Connected clients now is", len(clientConnected))
	for {
	}

	//for {
	//	mt, message, err := wsConn.ReadMessage()
	//	if err != nil {
	//		log.Println("read:", err)
	//		break
	//	}
	//
	//	log.Printf("recv: %d - %s\n", mt, string(message))
	//	err = c.WriteMessage(mt, message)
	//	if err != nil {
	//		log.Println("write:", err)
	//		break
	//	}
	//}
}
