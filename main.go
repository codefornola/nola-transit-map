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

	// Time between fetches to Clever Devices Bustime server.
	scraperFetchInterval = 10 * time.Second

	// Use in place of Clever Devices URL when in DEV mode.
	mockCleverDevicesUrl = "http://localhost:8081/getvehicles"

	// Clever Devices API URL: http://[host:port]/bustime/api/v3/getvehicles
	// http://ride.smtd.org/bustime/apidoc/docs/DeveloperAPIGuide3_0.pdf
	cleverDevicesUrlFormatter = "https://%s/bustime/api/v3/getvehicles"

	// Append to Clever Devices base url (above).
	// tmres=m -> time resolution: minute.
	// rtpidatafeed=bustime -> specify the Bustime data feed.
	// format=json -> respond with json (as opposed to XML).
	cleverDevicesVehicleQueryFormatter = "%s?key=%s&tmres=m&rtpidatafeed=bustime&format=json"
)

var (
	addr     = flag.String("addr", ":8080", "http service address")
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	DEV = false
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
	BaseUrl  string        `yaml:"url"`
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

	baseUrl := fmt.Sprintf(cleverDevicesUrlFormatter, ip)
	if DEV {
		baseUrl = mockCleverDevicesUrl
	}
	config := &Config{
		BaseUrl:  baseUrl,
		Interval: scraperFetchInterval,
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
		result, err := s.fetch()
		if err != nil {
			log.Printf(
				"ERROR: Scraper: Could not reach the Clever Devices server. Trying again in %d seconds. \n",
				int(scraperFetchInterval.Seconds()),
			)
			time.Sleep(scraperFetchInterval)
			continue
		}
		log.Printf("Found %d vehicles\n", len(result.Vehicles))
		vs <- result.Vehicles
		time.Sleep(s.config.Interval)
	}
}

func (v *Scraper) fetch() (*BustimeData, error) {
	key := v.config.Key
	baseURL := v.config.BaseUrl
	url := fmt.Sprintf(cleverDevicesVehicleQueryFormatter, baseURL, key)
	log.Println("Scraper URL:", url)
	resp, err := v.client.Get(url)
	if err != nil {
		log.Println("ERROR: Scraper response:", err)
		return nil, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Fatal("ERROR: Scraper response reader:", readErr)
	}

	result := &BustimeResponse{}
	json.Unmarshal(body, result)

	return &result.Data, nil
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
	scraper := NewScraper()
	defer scraper.Close()
	log.Println("Starting scraper")
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
	http.HandleFunc("/sse", s.serveSSE)

	log.Println("Starting server")
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}

func (s *Server) writeVehicles(w http.ResponseWriter, vehicles []Vehicle) error {
	if len(vehicles) > 0 {
		payload, err := json.Marshal(vehicles)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "data: %s\n\n", payload)
	} else {
		log.Println("No Vehicles to write")
	}
	return nil
}

func (s *Server) serveSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	vehicleChan := make(VehicleChannel)
	s.broadcaster.Register(vehicleChan)
	log.Println("Sending cached vehicles")
	s.writeVehicles(w, s.broadcaster.vehicles)

	for vehicles := range vehicleChan {
		err := s.writeVehicles(w, vehicles)
		if err != nil {
			break
		}
	}
	log.Println("SSE connection closed")
}

func main() {
	if _, exists := os.LookupEnv("DEV"); exists {
		DEV = true
		log.Println("Set to DEV mode.")
	}

	server := NewServer()
	server.Start()
}
