package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Bid struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

var manager = AuctionManager{}
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	clientID := template.URLQueryEscaper(r.URL.Path[len("/ws/"):])
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}
	defer ws.Close()
	manager.connect(ws)
	log.Printf("Client #%s joined the auction", clientID)

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			manager.disconnect(ws)
			log.Printf("Client #%s left the auction", clientID)
			break
		}
		var bid Bid
		if err := json.Unmarshal(message, &bid); err != nil {
			log.Printf("Unmarshal error: %v", err)
			continue
		}
		manager.handleBid(bid)
	}
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "client.html")
	})
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})
	http.HandleFunc("/ws/", wsEndpoint)
	http.HandleFunc("/close", func(w http.ResponseWriter, r *http.Request) {
		manager.closeAuction()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Auction closed",
			"bid":     manager.currentBid,
		})
	})

	log.Fatal(http.ListenAndServe(":8000", nil))
}
