package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write the file to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

var (
	addr = flag.String("addr", ":8080", "http service address")
)

type VehicleChannel = chan []Vehicle

type VehicleBroadcaster struct {
	incoming  VehicleChannel
	vehicles  []Vehicle
	receivers map[VehicleChannel]bool
}

func NewVehicleBroadCaster() *VehicleBroadcaster {
	return &VehicleBroadcaster{
		incoming:  make(VehicleChannel),
		receivers: make(map[VehicleChannel]bool),
	}
}

func (b *VehicleBroadcaster) Register(c VehicleChannel) {
	b.receivers[c] = true
}

func (b *VehicleBroadcaster) Unregister(c VehicleChannel) {
	delete(b.receivers, c)
}

func (b *VehicleBroadcaster) Start() {
	//config := bustime.GetConfig()
	scraper := NewScraper()
	defer scraper.Close()
	log.Println("start sraper")
	go scraper.Start(b.incoming)
	b.broadcast()
}

func (b *VehicleBroadcaster) broadcast() {
	for vs := range b.incoming {
		log.Println("Caching Vehicles")
		b.vehicles = vs
		log.Printf("%d listeners \n", len(b.receivers))
		for r, _ := range b.receivers {
			log.Println("Broadcasting Vehicles")
			select {
			case r <- vs:
			default:
				log.Println("Closing")
				close(r)
				b.Unregister(r)
			}
		}
		log.Println("Done Broadcasting")
	}
}

type Server struct {
	broadcaster *VehicleBroadcaster
}

func NewServer() *Server {
	return &Server{
		broadcaster: NewVehicleBroadCaster(),
	}
}

func (s *Server) Start() {
	go s.broadcaster.Start()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/index.html")
	})

	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/public/", http.StripPrefix("/public/", fs))

	// Handle websocket connection
	http.HandleFunc("/ws", s.serveWs)

	log.Println("Starting server")
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}

func (s *Server) reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	//ws.SetReadDeadline(time.Now().Add(pongWait))
	//ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (s *Server) writeVehicles(ws *websocket.Conn, vehicles []Vehicle) error {
	if len(vehicles) > 0 {
		payload, err := json.Marshal(vehicles)
		if err != nil {
			return err
		}
		ws.SetWriteDeadline(time.Now().Add(writeWait))
		err = ws.WriteMessage(websocket.TextMessage, payload)
		if err != nil {
			return err
		}
	} else {
		log.Println("No Vehicles to write")
	}
	return nil
}

func (s *Server) writer(ws *websocket.Conn) {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		ws.Close()
	}()

	vehicleChan := make(VehicleChannel)
	s.broadcaster.Register(vehicleChan)
	log.Println("Sending cached vehicles")
	s.writeVehicles(ws, s.broadcaster.vehicles)

	for vehicles := range vehicleChan {
		err := s.writeVehicles(ws, vehicles)
		if err != nil {
			break
		}
	}
	log.Println("Got close. Stopping WS writer")
}

func (s *Server) serveWs(w http.ResponseWriter, r *http.Request) {
	log.Println("serving ws")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}
	go s.writer(ws)
	s.reader(ws)
}

func NewScraper() *Scraper {
	api_key, ok := os.LookupEnv("CLEVER_DEVICES_KEY")
	if !ok {
		panic("Need to set environment variable CLEVER_DEVICES_KEY. Try `make run CLEVER_DEVICES_KEY=thekey`. Get key from Ben on slack")
	}
	ip, ok := os.LookupEnv("CLEVER_DEVICES_IP")
	if !ok {
		panic("Need to set environment variable CLEVER_DEVICES_KEY. Try `make run CLEVER_DEVICES_KEY=thekey`. Get key from Ben on slack")
	}
}

func main() {
	server := NewServer()
	server.Start()
}
