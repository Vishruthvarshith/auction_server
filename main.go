package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Bid struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type AuctionManager struct {
	activeBidders []*websocket.Conn
	currentBid    Bid
	lock          sync.Mutex
}

func (am *AuctionManager) connect(ws *websocket.Conn) {
	am.lock.Lock()
	am.activeBidders = append(am.activeBidders, ws)
	am.lock.Unlock()
}

func (am *AuctionManager) disconnect(ws *websocket.Conn) {
	am.lock.Lock()
	for i, conn := range am.activeBidders {
		if conn == ws {
			am.activeBidders = append(am.activeBidders[:i], am.activeBidders[i+1:]...)
			break
		}
	}
	am.lock.Unlock()
}

func (am *AuctionManager) broadcast(message string) {
	am.lock.Lock()
	for _, conn := range am.activeBidders {
		err := conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Printf("broadcast error: %v", err)
		}
	}
	am.lock.Unlock()
}

func (am *AuctionManager) handleBid(bid Bid, ws *websocket.Conn) {
	if bid.Value > am.currentBid.Value {
		am.currentBid = bid
		fmt.Println(bid)
		bidMsg, _ := json.Marshal(map[string]interface{}{
			"event": "new_bid",
			"bid":   am.currentBid,
		})
		am.broadcast(string(bidMsg))
	}
}

func (am *AuctionManager) closeAuction() {
	bidMsg, _ := json.Marshal(map[string]interface{}{
		"event": "auction_end",
		"bid":   am.currentBid,
	})
	am.broadcast(string(bidMsg))
	fmt.Println(am.currentBid)
	am.activeBidders = []*websocket.Conn{}
	am.currentBid = Bid{Name: "", Value: 0.0}
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
		manager.handleBid(bid, ws)
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
