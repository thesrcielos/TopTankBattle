package actions

import (
	"encoding/json"
	"log"

	"github.com/thesrcielos/TopTankBattle/internal/game"
	"github.com/thesrcielos/TopTankBattle/websocket/message"
)

func HandleGameStart(playerId string, msg message.Message, gameService game.GameService) {
	var gameStartPayload message.GameStartPayload
	if err := json.Unmarshal(msg.Payload, &gameStartPayload); err != nil {
		log.Println("Error decoding: ", err)
		return
	}

	gameService.StartGame(playerId, gameStartPayload.RoomId, false)
}
