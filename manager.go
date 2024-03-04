package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type AuctionManager struct {
	activeBidders []*websocket.Conn
	currentBid    Bid
	lock          sync.Mutex
}

func (am *AuctionManager) connect(ws *websocket.Conn) {
	am.lock.Lock()
	defer am.lock.Unlock()
	am.activeBidders = append(am.activeBidders, ws)
}

func (am *AuctionManager) disconnect(ws *websocket.Conn) {
	am.lock.Lock()
	defer am.lock.Unlock()
	for i, conn := range am.activeBidders {
		if conn == ws {
			am.activeBidders = append(am.activeBidders[:i], am.activeBidders[i+1:]...)
			break
		}
	}
}

func (am *AuctionManager) broadcast(message string) {
	am.lock.Lock()
	defer am.lock.Unlock()
	for _, conn := range am.activeBidders {
		err := conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Printf("broadcast error: %v", err)
		}
	}
}

func (am *AuctionManager) handleBid(bid Bid) {
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
