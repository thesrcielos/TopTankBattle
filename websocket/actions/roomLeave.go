package actions

import (
	"encoding/json"
	"log"

	"github.com/thesrcielos/TopTankBattle/websocket/message"
)

func HandleRoomLeave(playerId string, msg message.Message) {
	var roomLeave message.RoomPlayerLeave
	if err := json.Unmarshal(msg.Payload, &roomLeave); err != nil {
		log.Println("Error decoding: ", err)
		return
	}
	/*room, err := game.LeaveRoom(playerId)
	if err != nil {
		log.Println(err)
		return
	}

	response := transport.OutgoingMessage{
		Type: "ROOM_LEAVE",
		Payload: message.RoomPlayerLeave{
			Player: roomLeave.Player,
			Room:   roomLeave.Room,
		},
	}

	for _, p := range room.Team1 {
		transport.SendToPlayer(p.ID, response)
	}

	for _, p := range room.Team2 {
		transport.SendToPlayer(p.ID, response)
	}*/
}
