package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/thesrcielos/TopTankBattle/internal/game/state"
	"github.com/thesrcielos/TopTankBattle/pkg/db"
	"github.com/thesrcielos/TopTankBattle/websocket/transport"
)

var subs = make(map[string]*redis.PubSub)
var instanceID = getEnv("INSTANCE_ID", uuid.New().String())

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
func PublishToRoom(roomID string, payload string) {
	err := db.Rdb.Publish(ctx, "room:"+roomID, payload).Err()
	if err != nil {
		log.Println("Error publishing to room:", err)
	}
}

func SubscribeToRoom(roomID string) error {
	if _, exists := subs[roomID]; exists {
		return nil
	}

	sub := db.Rdb.Subscribe(ctx, "room:"+roomID)

	_, err := sub.Receive(ctx)
	if err != nil {
		log.Println("error subscribing to room %s: %w", roomID, err)
		return fmt.Errorf("error subscribing to room %s: %w", roomID, err)
	}

	ch := sub.Channel()
	subs[roomID] = sub

	log.Printf("Subscribed to room %s", roomID)
	go func() {
		for msg := range ch {
			SendReceivedMessage(msg.Payload)
		}
	}()

	return nil
}

func UnsubscribeFromRoom(roomID string) error {
	sub := subs[roomID]
	if err := sub.Unsubscribe(ctx, "room:"+roomID); err != nil {
		return fmt.Errorf("error unsubscribing from room %s: %w", roomID, err)
	}

	delete(subs, roomID)
	return nil
}

func SendReceivedMessage(messageEncoded string) {
	var message GameMessage
	if err := json.Unmarshal([]byte(messageEncoded), &message); err != nil {
		log.Println("Error decoding message:", err)
		return
	}
	if message.Type == "GAME_MOVE" {
		payloadBytes, _ := json.Marshal(message.Payload)
		var move MovePlayerMessage
		json.Unmarshal(payloadBytes, &move)
		updateGamePlayerState(move.PlayerId, move.Position)
		return
	}
	if message.Type == "GAME_SHOOT" {
		payloadBytes, _ := json.Marshal(message.Payload)
		var bullet state.Bullet
		json.Unmarshal(payloadBytes, &bullet)
		updateGameBullets(bullet)
		return
	}
	if message.Type == "GAME_START_INFO" {
		payloadBytes, _ := json.Marshal(message.Payload)
		var info GameInfo
		err := json.Unmarshal(payloadBytes, &info)
		if err != nil {
			log.Println("Error decoding game start info:", err)
			return
		}
		if info.Instance != instanceID {
			AttemptLeadership(info.RoomId)
		}
		return
	}
	msg := transport.OutgoingMessage{
		Type:    message.Type,
		Payload: message.Payload,
	}

	for _, playerId := range message.Users {
		transport.SendToPlayer(playerId, msg)
	}
}

func updateGamePlayerState(playerId string, position state.Position) {
	player := state.GetPlayer(playerId)
	if player == nil || player.GameState == nil {
		return
	}
	game := player.GameState
	game.GameMu.Lock()
	game.Players[playerId].Position = position
	game.GameMu.Unlock()
}

func updateGameBullets(bullet state.Bullet) {
	player := state.GetPlayer(bullet.OwnerId)
	if player == nil || player.GameState == nil {
		return
	}
	game := player.GameState
	game.GameMu.Lock()
	game.Bullets[bullet.ID] = &bullet
	game.GameMu.Unlock()
}

type MovePlayerMessage struct {
	PlayerId string         `json:"playerId"`
	Position state.Position `json:"position"`
}

type GameInfo struct {
	Instance string `json:"instance"`
	RoomId   string `json:"roomId"`
}

func tryToBecomeLeader(roomID string) bool {
	key := fmt.Sprintf("leader:%s", roomID)
	ok, err := db.Rdb.SetNX(ctx, key, instanceID, 5000*time.Millisecond).Result()
	if err != nil {
		return false
	}
	return ok
}

func saveGameStateToRedis(gameState *state.GameState) {
	roomID := gameState.RoomId
	for _, b := range gameState.Bullets {
		key := fmt.Sprintf("room:%s:bullet:%s", roomID, b.ID)
		db.Rdb.HSet(ctx, key, map[string]interface{}{
			"x":       b.Position.X,
			"y":       b.Position.Y,
			"angle":   b.Position.Angle,
			"speed":   b.Speed,
			"ownerId": b.OwnerId,
		})
		db.Rdb.Expire(ctx, key, 10*time.Second)
	}

	for _, p := range gameState.Players {
		key := fmt.Sprintf("room:%s:player:%s", roomID, p.ID)
		db.Rdb.HSet(ctx, key, map[string]interface{}{
			"x":     p.Position.X,
			"y":     p.Position.Y,
			"angle": p.Position.Angle,
		})
	}
}

func restoreGameStateFromRedis(roomID string) *state.GameState {
	gameState := state.GameState{}

	keys, _ := db.Rdb.Keys(ctx, fmt.Sprintf("room:%s:bullet:*", roomID)).Result()
	for _, key := range keys {
		vals, _ := db.Rdb.HGetAll(ctx, key).Result()
		b := state.Bullet{
			ID: key[len(fmt.Sprintf("room:%s:bullet:", roomID)):],
			Position: state.Position{
				X:     parseFloat(vals["x"]),
				Y:     parseFloat(vals["y"]),
				Angle: parseFloat(vals["angle"]),
			},
			Speed:   parseFloat(vals["speed"]),
			OwnerId: vals["ownerId"],
		}
		gameState.Bullets[b.ID] = &b
	}

	keys, _ = db.Rdb.Keys(ctx, fmt.Sprintf("room:%s:player:*", roomID)).Result()
	for _, key := range keys {
		vals, _ := db.Rdb.HGetAll(ctx, key).Result()
		p := state.PlayerState{
			ID: key[len(fmt.Sprintf("room:%s:player:", roomID)):],
			Position: state.Position{
				X:     parseFloat(vals["x"]),
				Y:     parseFloat(vals["y"]),
				Angle: parseFloat(vals["angle"]),
			},
		}
		gameState.Players[p.ID] = &p
	}

	return &gameState
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func RenewLeadership(roomID string, expiration time.Duration) (bool, error) {
	key := fmt.Sprintf("leader:%s", roomID)

	currentLeader, err := db.Rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		fmt.Println("No current leader found")
		if tryToBecomeLeader(roomID) {
			return true, nil
		}
		return false, nil
	} else if err != nil {
		return false, err
	}

	if currentLeader == instanceID {
		ok, err := db.Rdb.Expire(ctx, key, expiration).Result()
		if !ok || err != nil {
			return false, errors.New("failed to renew leadership")
		}
		return true, nil
	}

	return false, nil
}
