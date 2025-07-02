package router

import (
	"log"

	"github.com/thesrcielos/TopTankBattle/websocket/actions"
	"github.com/thesrcielos/TopTankBattle/websocket/message"
)

var handlers = map[string]func(playerId string, payload message.Message){
	"ROOM_DELETION": actions.HandleRoomDeletion,
	"ROOM_LEAVE":    actions.HandleRoomLeave,
	"MOVE":          actions.HandleMove,
	"SHOOT":         actions.HandleShoot,
}

func RouteMessage(playerId string, msg message.Message) {
	if handler, ok := handlers[msg.Type]; ok {
		handler(playerId, msg)
	} else {
		log.Println("Tipo de mensaje desconocido:", msg.Type)
	}
}
