package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/thesrcielos/TopTankBattle/internal/game/state"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

func WebSocketHandler(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return err
	}

	_, rawID, err := ws.ReadMessage()
	if err != nil {
		log.Println("Failed to read player ID:", err)
		return err
	}

	pId := string(rawID)
	log.Printf("Player connected: %s", pId)

	state.RegisterPlayer(pId, ws)
	go listenPlayerMessages(pId, ws)

	return nil
}
