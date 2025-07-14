package actions

import (
	"encoding/json"

	"github.com/thesrcielos/TopTankBattle/internal/game"
	"github.com/thesrcielos/TopTankBattle/websocket/message"
)

func HandleRoomKick(playerId string, msg message.Message) {
	var kickRequest message.RoomKickRequestPayload
	if err := json.Unmarshal(msg.Payload, &kickRequest); err != nil {
		return
	}
	game.KickPlayerFromRoom(playerId, kickRequest.RoomId, kickRequest.PlayerId)
}
