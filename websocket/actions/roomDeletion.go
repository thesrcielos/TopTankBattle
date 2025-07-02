package actions

import (
	"encoding/json"
	"log"

	"github.com/thesrcielos/TopTankBattle/internal/game"
	"github.com/thesrcielos/TopTankBattle/websocket/message"
	"github.com/thesrcielos/TopTankBattle/websocket/transport"
)

func HandleRoomDeletion(playerId string, msg message.Message) {
	var roomDeletion message.RoomDeletionRequestPayload
	if err := json.Unmarshal(msg.Payload, &roomDeletion); err != nil {
		log.Println("Error decoding: ", err)
		return
	}
	players, err := game.DeleteRoom(playerId, roomDeletion.Room)
	if err != nil {
		log.Println(err)
		return
	}

	response := transport.OutgoingMessage{
		Type: "ROOM_DELETION",
		Payload: message.RoomDelete{
			RoomId: roomDeletion.Room,
		},
	}

	for _, p := range *players {
		transport.SendToPlayer(p.ID, response)
	}
}
