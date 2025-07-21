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

	"github.com/thesrcielos/TopTankBattle/websocket/transport"
)

var instanceID = getEnv("INSTANCE_ID", uuid.New().String())

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func NewGameStateRepository(leaderElector LeaderElector, db *redis.Client) *RedisGameStateRepository {
	return &RedisGameStateRepository{
		LeaderElector: leaderElector,
		db:            db,
	}
}

type GameStateRepository interface {
	PublishToRoom(payload string)
	SubscribeMessages() error
	SendReceivedMessage(messageEncoded string)
	TryToBecomeLeader(roomID string) bool
	SaveGameState(gameState *state.GameState)
	RestoreGameState(roomID string) *state.GameState
	RenewLeadership(roomID string, expiration time.Duration) (bool, error)
	UpdateGamePlayerState(playerId string, position state.Position)
	UpdateGameBullets(bullet state.Bullet)
}

type RedisGameStateRepository struct {
	LeaderElector LeaderElector
	db            *redis.Client
}

func (r *RedisGameStateRepository) PublishToRoom(payload string) {
	err := r.db.Publish(ctx, "messages", payload).Err()
	if err != nil {
		log.Println("Error publishing to room:", err)
	}
}

func (r *RedisGameStateRepository) SubscribeMessages() error {
	sub := r.db.Subscribe(ctx, "messages")
	_, err := sub.Receive(ctx)
	if err != nil {
		log.Println("error subscribing", err)
		return fmt.Errorf("error subscribing %w", err)
	}

	ch := sub.Channel()

	log.Printf("Subscribed to messages channel")
	go func() {
		for msg := range ch {
			r.SendReceivedMessage(msg.Payload)
		}
	}()

	return nil
}

func (r *RedisGameStateRepository) SendReceivedMessage(messageEncoded string) {
	var message GameMessage
	if err := json.Unmarshal([]byte(messageEncoded), &message); err != nil {
		log.Println("Error decoding message:", err)
		return
	}
	fmt.Println("Received message:", message.Type, "for players:", message.Users)
	if message.Type == "GAME_MOVE" {
		payloadBytes, _ := json.Marshal(message.Payload)
		var move MovePlayerMessage
		json.Unmarshal(payloadBytes, &move)
		r.UpdateGamePlayerState(move.PlayerId, move.Position)
		return
	}
	if message.Type == "GAME_SHOOT" {
		payloadBytes, _ := json.Marshal(message.Payload)
		var bullet state.Bullet
		json.Unmarshal(payloadBytes, &bullet)
		r.UpdateGameBullets(bullet)
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
			go r.LeaderElector.AttemptLeadership(info.RoomId)
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

func (r *RedisGameStateRepository) UpdateGamePlayerState(playerId string, position state.Position) {
	player := state.GetPlayer(playerId)
	if player == nil || player.GameState == nil {
		return
	}
	game := player.GameState
	game.GameMu.Lock()
	game.Players[playerId].Position = position
	game.GameMu.Unlock()
}

func (r *RedisGameStateRepository) UpdateGameBullets(bullet state.Bullet) {
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

func (r *RedisGameStateRepository) TryToBecomeLeader(roomID string) bool {
	key := fmt.Sprintf("leader:%s", roomID)
	ok, err := r.db.SetNX(ctx, key, instanceID, 5000*time.Millisecond).Result()
	if err != nil {
		return false
	}
	return ok
}

func (r *RedisGameStateRepository) SaveGameState(gameState *state.GameState) {
	roomID := gameState.RoomId
	for _, b := range gameState.Bullets {
		key := fmt.Sprintf("room:%s:bullet:%s", roomID, b.ID)
		r.db.HSet(ctx, key, map[string]interface{}{
			"x":       b.Position.X,
			"y":       b.Position.Y,
			"angle":   b.Position.Angle,
			"speed":   b.Speed,
			"ownerId": b.OwnerId,
		})
		r.db.Expire(ctx, key, 10*time.Second)
	}

	for _, p := range gameState.Players {
		key := fmt.Sprintf("room:%s:player:%s", roomID, p.ID)
		r.db.HSet(ctx, key, map[string]interface{}{
			"x":     p.Position.X,
			"y":     p.Position.Y,
			"angle": p.Position.Angle,
		})
	}

	for _, f := range gameState.Fortresses {
		key := fmt.Sprintf("room:%s:fortress:%s", roomID, f.ID)
		r.db.HSet(ctx, key, map[string]interface{}{
			"x":      f.Position.X,
			"y":      f.Position.Y,
			"health": f.Health,
			"team1":  f.Team1,
		})
	}
}

func (r *RedisGameStateRepository) RestoreGameState(roomID string) *state.GameState {
	gameState := state.GameState{
		RoomId:     roomID,
		Bullets:    make(map[string]*state.Bullet),
		Players:    make(map[string]*state.PlayerState),
		Fortresses: make([]*state.Fortress, 0),
	}

	keys, _ := r.db.Keys(ctx, fmt.Sprintf("room:%s:bullet:*", roomID)).Result()
	for _, key := range keys {
		vals, _ := r.db.HGetAll(ctx, key).Result()
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

	keys, _ = r.db.Keys(ctx, fmt.Sprintf("room:%s:player:*", roomID)).Result()
	for _, key := range keys {
		vals, _ := r.db.HGetAll(ctx, key).Result()
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

	keys, _ = r.db.Keys(ctx, fmt.Sprintf("room:%s:fortress:*", roomID)).Result()
	for _, key := range keys {
		vals, _ := r.db.HGetAll(ctx, key).Result()
		f := state.Fortress{
			ID: key[len(fmt.Sprintf("room:%s:fortress:", roomID)):],
			Position: state.Position{
				X:     parseFloat(vals["x"]),
				Y:     parseFloat(vals["y"]),
				Angle: 0, // Fortress does not have an angle
			},
			Health: parseInt(vals["health"]),
			Team1:  vals["team1"] == "true",
		}
		gameState.Fortresses = append(gameState.Fortresses, &f)
	}
	return &gameState
}

func parseInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}
func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func (r *RedisGameStateRepository) RenewLeadership(roomID string, expiration time.Duration) (bool, error) {
	key := fmt.Sprintf("leader:%s", roomID)

	currentLeader, err := r.db.Get(ctx, key).Result()
	if err == redis.Nil {
		fmt.Println("No current leader found")
		if r.TryToBecomeLeader(roomID) {
			return true, nil
		}
		return false, nil
	} else if err != nil {
		return false, err
	}

	if currentLeader == instanceID {
		ok, err := r.db.Expire(ctx, key, expiration).Result()
		if !ok || err != nil {
			return false, errors.New("failed to renew leadership")
		}
		return true, nil
	}

	return false, nil
}
