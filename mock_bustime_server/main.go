package main

import (
	"log"
	"net/http"
)

const port = ":8081"

func main() {
	http.HandleFunc(
		"/getvehicles",
		func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./mock_vehicles.json")
			log.Println("GET /getvehicles :: Served Vehicles")
		},
	)
	log.Printf("\n\nMock bustime server is running at: http://localhost%s \n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
