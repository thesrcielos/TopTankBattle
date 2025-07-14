package actions

import (
	"encoding/json"
	"log"

	"github.com/thesrcielos/TopTankBattle/internal/game"
	"github.com/thesrcielos/TopTankBattle/internal/game/state"
	"github.com/thesrcielos/TopTankBattle/websocket/message"
)

func HandleMove(playerId string, msg message.Message) {
	var movePayload message.GameMovePayload
	if err := json.Unmarshal(msg.Payload, &movePayload); err != nil {
		log.Println("Error decoding", err)
		return
	}
	position := state.Position{
		X:     movePayload.X,
		Y:     movePayload.Y,
		Angle: movePayload.Angle,
	}
	game.MovePlayer(playerId, position)
}
