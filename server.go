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

type ServerConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	EndPoint string `json:"end_point"`
}

func init() {
	flag.StringVar(&configFile, "configFile", JSON_FILE, "Type your config file for parse them")
	flag.Parse()

	//if _, err := os.Stat("/var/log/msg-service"); os.IsNotExist(err) {
	//	os.MkdirAll("/var/log/msg-service", 0775)
	//}
	//// Write log data into console and log file
	//f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	//if err != nil {
	//	log.Fatalf("error opening file: %v", err)
	//}
	//
	//wrt := io.MultiWriter(os.Stdout, f)
	//log.SetOutput(wrt)
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
	r.Use(ml.GetClients)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Get(serverConfig.EndPoint, listClientSocket)

	log.Println("Start listen server at ", fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port))
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port), r)
	if err != nil {
		log.Println("Error to start listen server", err)
	}
}

func listClientSocket(writer http.ResponseWriter, request *http.Request) {
	wsConn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Println("Cant send upgrade status for client", err)
		return
	}

	defer wsConn.Close()

	var clientContext = request.Context().Value("client")
	log.Println("Request params", clientContext)
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

func (cl *ml.Clients) Add() {
	log.Println("add new client")
}
