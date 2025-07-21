package router

import (
	"log"

	"github.com/thesrcielos/TopTankBattle/internal/game"
	"github.com/thesrcielos/TopTankBattle/websocket/actions"
	"github.com/thesrcielos/TopTankBattle/websocket/message"
)

var handlers = map[string]func(playerId string, payload message.Message, game game.GameService){
	"MOVE":       actions.HandleMove,
	"SHOOT":      actions.HandleShoot,
	"GAME_START": actions.HandleGameStart,
}

func RouteMessage(playerId string, msg message.Message, GameService game.GameService) {
	if handler, ok := handlers[msg.Type]; ok {
		handler(playerId, msg, GameService)
	} else {
		log.Println("Tipo de mensaje desconocido:", msg.Type)
	}
}
