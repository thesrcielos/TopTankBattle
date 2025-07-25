package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thesrcielos/TopTankBattle/internal/game"
	"github.com/thesrcielos/TopTankBattle/internal/game/state"
	"github.com/thesrcielos/TopTankBattle/websocket/message"
	"github.com/thesrcielos/TopTankBattle/websocket/router"
)

var RoomService *game.RoomService

func listenPlayerMessages(playerId string, conn *websocket.Conn) {
	defer func() {
		state.UnregisterPlayerDelayed(playerId, 20*time.Second, RoomService.LeaveRoom)
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

		router.RouteMessage(playerId, msg, GameService)
	}
}
