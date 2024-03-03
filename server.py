from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.responses import HTMLResponse

from pydantic import BaseModel
import json

app = FastAPI()

with open("client.html", "r") as file:
    client_html = file.read()


class Bid(BaseModel):
    name: str
    value: float


class AuctionManager:
    def __init__(self):
        self.active_bidders: list[WebSocket] = []
        self.current_bid = Bid(name='', value=0.0)

    async def connect(self, websocket: WebSocket):
        await websocket.accept()
        self.active_bidders.append(websocket)

    def disconnect(self, websocket: WebSocket):
        self.active_bidders.remove(websocket)

    async def broadcast(self, message: str):
        for connection in self.active_bidders:
            await connection.send_text(message)

    async def handle_bid(self, bid: Bid, websocket: WebSocket):
        if bid.value > self.current_bid.value:
            self.current_bid = bid
            print(bid)
            await self.broadcast(json.dumps({
                "event": "new_bid",
                "bid": self.current_bid.dict(),
            }))

    async def close_auction(self):
        await self.broadcast(json.dumps({
            "event": "auction_end",
            "bid": self.current_bid.dict(),
        }))
        print(self.current_bid)
        self.active_bidders = []
        self.current_bid = Bid(name='', value=0.0)


manager = AuctionManager()


@app.get("/")
async def get():
    return HTMLResponse(client_html)


@app.get("/ping")
async def pong():
    return {"message": "pong"}


@app.websocket("/ws/{client_id}")
async def websocket_endpoint(websocket: WebSocket, client_id: int):
    await manager.connect(websocket)
    try:
        while True:
            data = await websocket.receive_text()
            bid = Bid.parse_raw(data)  # Parse the JSON data into a Bid object
            await manager.handle_bid(bid, websocket)
    except WebSocketDisconnect:
        manager.disconnect(websocket)
        print(f"Client #{client_id} left the auction")


@app.get("/close")
async def close_auction():
    await manager.close_auction()
    return {
        "message": "Auction closed",
        "bid": manager.current_bid.dict()
    }
