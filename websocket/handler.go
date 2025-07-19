package websocket

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/thesrcielos/TopTankBattle/internal/game/state"
	"github.com/thesrcielos/TopTankBattle/pkg/db"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

func WebSocketHandler(c echo.Context) error {
	tokenString := c.QueryParam("token")

	userID, err := ValidateJWT(tokenString)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return err
	}

	ctx := context.Background()
	val, err := db.Rdb.Get(ctx, userID).Result()
	fmt.Println("User room ID:", val)
	if err == redis.Nil {
		ws.WriteMessage(websocket.TextMessage, []byte("User room not found"))
		ws.Close()
		log.Printf("User room not found for user %s", userID)
		return nil
	} else if err != nil {
		ws.WriteMessage(websocket.TextMessage, []byte("Error retrieving user room"))
		ws.Close()
		log.Printf("Error retrieving user room for user %s", userID)
		return nil
	}
	log.Printf("Player connected: %s", userID)
	state.RegisterPlayer(userID, val, ws)
	go listenPlayerMessages(userID, ws)

	return nil
}

func ValidateJWT(tokenString string) (string, error) {
	if tokenString == "" {
		return "", errors.New("Empty token")
	}

	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Invalid token")
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil || !token.Valid {
		return "", fmt.Errorf("Invalid token: %v", err)
	}

	fmt.Println("Token claims:", claims)
	userID, ok := claims["id"].(float64)

	if !ok {
		return "", errors.New("user_id not found in token claims")
	}

	return strconv.Itoa(int(userID)), nil
}
