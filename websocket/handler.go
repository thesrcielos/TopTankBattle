package websocket

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"context"
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
	if err == redis.Nil {
		return echo.NewHTTPError(http.StatusBadRequest, "User room not found")
	} else if err != nil {
		return errors.New("Error retrieving user data from Redis")
	}
	log.Printf("Player connected: %s", userID)
	state.RegisterPlayer(userID, val, ws)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			err := ws.WriteMessage(websocket.PingMessage, nil)
			conn.Close()
			if err != nil{
				log.Printf("Error sending ping %s", err)
			}
		}
		time.Sleep(30 * time.Second)
	}()
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
