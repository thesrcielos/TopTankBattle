package websocket

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
	"github.com/thesrcielos/TopTankBattle/internal/game/state"
	"github.com/thesrcielos/TopTankBattle/websocket/message"
	"github.com/thesrcielos/TopTankBattle/websocket/router"
)

func listenPlayerMessages(playerId string, conn *websocket.Conn) {
	defer func() {
		log.Printf("Player Disconnected: %s", playerId)
		state.UnregisterPlayer(playerId)
		conn.Close()
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		var msg message.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Println("Error decoding message:", err)
			continue
		}

		router.RouteMessage(playerId, msg)
	}
}
