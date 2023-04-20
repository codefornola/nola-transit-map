package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
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
	addr     = flag.String("addr", ":8080", "http service address")
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type VehicleTimestamp struct {
	time.Time
}

// UnmarshalJSON
// We need a special unmarshal method for this string timestamp. It's of the
// form "YYYYMMDD hh:mm"
func (t *VehicleTimestamp) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return err
	}

	// "YYYYMMDD hh:mm" https://pkg.go.dev/time#pkg-constants
	format := "20060102 15:04"
	time, err := time.ParseInLocation(format, s, loc)
	if err != nil {
		return err
	}

	t.Time = time
	return nil
}

// Vehicle represents an individual reading of a vehicle and it's location
// at that point in time
// Example:
//
//	{
//	  "vid": "155",
//	  "tmstmp": "20200827 11:51",
//	  "lat": "29.962149326173048",
//	  "lon": "-90.05214051918121",
//	  "hdg": "357",
//	  "pid": 275,
//	  "rt": "5",
//	  "des": "Saratoga at Canal",
//	  "pdist": 10122,
//	  "dly": false,
//	  "spd": 20,
//	  "tatripid": "3130339",
//	  "tablockid": "15",
//	  "zone": "",
//	  "srvtmstmp": "20200827 11:51",
//	  "oid": "445",
//	  "or": true,
//	  "rid": "501",
//	  "blk": 2102,
//	  "tripid": 982856020
//	}
type Vehicle struct {
	Vid        string           `json:"vid"`
	Tmstmp     VehicleTimestamp `json:"tmstmp"`
	SrvTimstmp VehicleTimestamp `json:"srvtmstmp"`
	Lat        float64          `json:"lat,string"`
	Lon        float64          `json:"lon,string"`
	Hdg        string           `json:"hdg"`
	Rt         string           `json:"rt"`
	Tatripid   string           `json:"tatripid"`
	Tablockid  string           `json:"tablockid"`
	Zone       string           `json:"zone"`
	Oid        string           `json:"oid"`
	Rid        string           `json:"rid"`
	Des        string           `json:"des"`
	Pdist      int              `json:"pdist"`
	Pid        int              `json:"pid"`
	Spd        int              `json:"spd"`
	Blk        int              `json:"blk"`
	Tripid     int              `json:"tripid"`
	Dly        bool             `json:"dly"`
	Or         bool             `json:"or"`
}

type BusErr struct {
	Rt  string `json:"rt"`
	Msg string `json:"msg"`
}

type BustimeData struct {
	Vehicles []Vehicle `json:"vehicle"`
	Errors   []BusErr  `json:"error"`
}

type BustimeResponse struct {
	Data BustimeData `json:"bustime-response"`
}

type Config struct {
	Key      string        `yaml:"key"`
	Interval time.Duration `yaml:"interval"`
	Url      string        `yaml:"url"`
}

type Scraper struct {
	client *http.Client
	config *Config
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

	config := &Config{
		Url:      fmt.Sprintf("http://%s/bustime/api/v3/getvehicles", ip),
		Interval: 10 * time.Second,
		Key:      api_key,
	}
	tr := &http.Transport{
		MaxIdleConnsPerHost: 1024,
		TLSHandshakeTimeout: 0 * time.Second,
	}
	client := &http.Client{Transport: tr}
	return &Scraper{
		client,
		config,
	}
}

func (s *Scraper) Start(vs chan []Vehicle) {
	for {
		result := s.fetch()
		log.Printf("Found %d vehicles\n", len(result.Vehicles))
		vs <- result.Vehicles
		time.Sleep(s.config.Interval)
	}
}

func (v *Scraper) fetch() *BustimeData {
	key := v.config.Key
	baseURL := v.config.Url
	url := fmt.Sprintf("%s?key=%s&tmres=m&rtpidatafeed=bustime&format=json", baseURL, key)
	resp, err := v.client.Get(url)
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Println(err)
	}
	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	result := &BustimeResponse{}
	json.Unmarshal(body, result)

	return &result.Data
}

func (v *Scraper) Close() {
	v.client.CloseIdleConnections()
}

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

func main() {
	server := NewServer()
	server.Start()
}
