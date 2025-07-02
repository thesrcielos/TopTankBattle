package game

import (
	"github.com/thesrcielos/TopTankBattle/websocket/transport"
)

type RoomPlayerJoin struct {
	Player Player `json:"player"`
}

func HandleRoomJoin(player *Player, room *Room) {
	messageResponse := transport.OutgoingMessage{
		Type: "ROOM_JOIN",
		Payload: RoomPlayerJoin{
			Player: *player,
		},
	}

	for _, p := range room.Team1 {
		if p.ID != player.ID {
			transport.SendToPlayer(p.ID, messageResponse)
		}
	}

	for _, p := range room.Team2 {
		if p.ID != player.ID {
			transport.SendToPlayer(p.ID, messageResponse)
		}
	}

}
