package main

import (
	"bitbucket.org/projectt_ct/websocket-service/server"
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
	"strings"
)

const (
	JSON_FILE = "./config.json"
	LOG_FILE  = "/var/log/websocket-service/websocket-service.log"
)

type ServerConfig struct {
	Host              string `json:"host"`
	Port              int    `json:"port"`
	UseSsl            bool   `json:"use_ssl"`
	SslCertificate    string `json:"ssl_certificate"`
	SslCertificateKey string `json:"ssl_certificate_key"`
}

type Config struct {
	Server ServerConfig `json:"server"`
}

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

	pool := server.NewClientPool()
	go pool.StartCollector()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// Routes list
	r.Get("/ws/{key}-{id}", func(writer http.ResponseWriter, request *http.Request) {
		listenPaymentWaitSocket(pool, writer, request)
	})
	r.Post("/publish", func(writer http.ResponseWriter, request *http.Request) {
		publishPaymentWaitChannel(pool, writer, request)
	})

	r.Get("/room/{room}/{key}/{id}", func(writer http.ResponseWriter, request *http.Request) {
		startListenSocket(pool, writer, request)
	})
	r.Post("/publish/{room}/{key}/{id}", func(writer http.ResponseWriter, request *http.Request) {
		startPublishToSocket(pool, writer, request)
	})

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

func listenPaymentWaitSocket(pool *server.Pool, writer http.ResponseWriter, request *http.Request) {
	ws, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Fatal("websocket conn failed", err)
	}

	key := chi.URLParam(request, "key")
	id, _ := strconv.Atoi(chi.URLParam(request, "id"))

	channelClientKey := fmt.Sprintf("%s_%d", key, id)
	var client = &server.Client{
		ChannelKey: channelClientKey,
		Pool: pool,
		Conn: ws,
		Send: make(chan []byte),
	}

	client.Pool.Register <- client
	go client.WritePump()
	go client.ReadPump()
}

func publishPaymentWaitChannel(pool *server.Pool, writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	if query["id"] == nil {
		log.Println("Cant publish to socket channel")
		return
	}

	if query["id"] != nil {
		strSplited := strings.Split(query["id"][0], "-")

		key := strSplited[0]
		id, _ := strconv.Atoi(strSplited[1])

		channelKey := fmt.Sprintf("%s_%d", key, id)
		if pool.Clients[channelKey] != nil {
			body, err := ioutil.ReadAll(request.Body)
			if err != nil {
				log.Println("Error to read body request", err)
				return
			}
			defer request.Body.Close()

			if pool.Clients[channelKey] != nil {
				for client, _ := range pool.Clients[channelKey] {
					if client.ChannelKey == channelKey {
						client.Send <- body
					}
				}
			}
		}
	}
}

func startListenSocket(pool *server.Pool, writer http.ResponseWriter, request *http.Request) {
	ws, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Fatal("websocket conn failed", err)
	}

	room := chi.URLParam(request, "room")
	key := chi.URLParam(request, "key")
	id, _ := strconv.Atoi(chi.URLParam(request, "id"))

	channelClientKey := fmt.Sprintf("%s_%s_%d", room, key, id)
	var client = &server.Client{
		ChannelKey: channelClientKey,
		Pool: pool,
		Conn: ws,
		Send: make(chan []byte),
	}

	client.Pool.Register <- client
	go client.WritePump()
	go client.ReadPump()
}

func startPublishToSocket(pool *server.Pool, writer http.ResponseWriter, request *http.Request) {
	room := chi.URLParam(request, "room")
	key := chi.URLParam(request, "key")
	id, _ := strconv.Atoi(chi.URLParam(request, "id"))

	log.Println("Key for published", fmt.Sprintf("%s_%s_%d", room, key, id))
	channelKey := fmt.Sprintf("%s_%s_%d", room, key, id)

	if pool.Clients[channelKey] != nil {
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			log.Println("Error to read body request", err)
			return
		}
		defer request.Body.Close()

		if pool.Clients[channelKey] != nil {
			for client, _ := range pool.Clients[channelKey] {
				if client.ChannelKey == channelKey {
					client.Send <- body
				}
			}
		}
	}
}