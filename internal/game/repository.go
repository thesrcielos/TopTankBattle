package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/thesrcielos/TopTankBattle/internal/apperrors"
	"github.com/thesrcielos/TopTankBattle/internal/user"
)

var ctx = context.Background()

type RedisRoomRepository struct {
	userRepository user.UserRepository
	db             *redis.Client
}

func NewRedisRoomRepository(userRepo user.UserRepository, db *redis.Client) *RedisRoomRepository {
	return &RedisRoomRepository{
		userRepository: userRepo,
		db:             db,
	}
}

type RoomRepository interface {
	SaveRoomRequest(*RoomRequest) (*Room, error)
	SaveRoom(*Room) error
	SavePlayerRoom(*PlayerRequest) error
	GetPlayerRoom(playerId string) (interface{}, error)
	DeletePlayerRoom(playerId string) error
	GetRoom(key string) (*Room, error)
	GetRooms(page, pageSize int) (*[]Room, error)
	CreatePlayer(id int) (*Player, error)
	AddPlayer(*PlayerRequest) (*Room, error)
	RemovePlayer(*PlayerRequest) (*Room, error)
	ChangeRoomOwner(roomId string, player Player) (*Room, error)
	DeleteRoom(id string) error
	PublishToRoom(payload string)
}

func (r *RedisRoomRepository) SaveRoomRequest(RoomRequest *RoomRequest) (*Room, error) {
	player, errDB := r.CreatePlayer(RoomRequest.Player)
	if errDB != nil {
		return nil, apperrors.NewAppError(500, "Error creating player", errDB)
	}

	key := uuid.New().String()[:8]
	room := &Room{
		ID:       key,
		Name:     RoomRequest.Name,
		Capacity: RoomRequest.Capacity,
		Players:  1,
		Team1:    []Player{*player},
		Team2:    []Player{},
		Host:     *player,
		Status:   "LOBBY",
	}

	if err := r.SaveRoom(room); err != nil {
		return nil, err
	}

	timestamp := float64(time.Now().Unix())
	if err := r.db.ZAdd(ctx, "rooms_id", redis.Z{Score: timestamp, Member: room.ID}).Err(); err != nil {
		return nil, apperrors.NewAppError(500, "Error saving room ID", err)
	}

	return room, nil
}

func (r *RedisRoomRepository) SaveRoom(room *Room) error {
	data, err := json.Marshal(room)
	if err != nil {
		return apperrors.NewAppError(500, "Error serializing room data", err)
	}

	if err := r.db.Set(ctx, room.ID, data, 0).Err(); err != nil {
		return apperrors.NewAppError(500, "Error saving room", err)
	}

	return nil
}

func (r *RedisRoomRepository) SavePlayerRoom(playerRequest *PlayerRequest) error {
	if err := r.db.Set(ctx, playerRequest.Player, playerRequest.Room, 0).Err(); err != nil {
		fmt.Println("Error saving player room:", err)
		return apperrors.NewAppError(500, "Error saving player room", err)
	}

	return nil
}

func (r *RedisRoomRepository) GetPlayerRoom(playerId string) (interface{}, error) {
	val, err := r.db.Get(ctx, playerId).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, apperrors.NewAppError(500, "Error getting player room", err)
	}

	return val, nil
}

func (r *RedisRoomRepository) DeletePlayerRoom(playerId string) error {
	if err := r.db.Del(ctx, playerId).Err(); err != nil {
		return apperrors.NewAppError(500, "Error deleting player room", err)
	}

	return nil
}

func (r *RedisRoomRepository) GetRoom(key string) (*Room, error) {
	val, err := r.db.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, apperrors.NewAppError(404, "Room not found", errors.New("room not found"))
	} else if err != nil {
		return nil, apperrors.NewAppError(500, "Error getting room", err)
	}
	var room Room
	if err := json.Unmarshal([]byte(val), &room); err != nil {
		return nil, apperrors.NewAppError(500, "Error unmarshalling room data", err)
	}

	return &room, nil
}

func (r *RedisRoomRepository) GetRooms(page, pageSize int) (*[]Room, error) {
	start := int64(page * pageSize)
	end := start + int64(pageSize) - 1

	roomIDs, err := r.db.ZRevRange(ctx, "rooms_id", start, end).Result()
	if err != nil {
		return nil, apperrors.NewAppError(500, "Error getting room IDs", err)
	}

	rooms := []Room{}
	for _, id := range roomIDs {
		room, err := r.GetRoom(id)
		if err != nil {
			return nil, apperrors.NewAppError(500, "Error getting room by ID", err)
		}
		rooms = append(rooms, *room)
	}

	return &rooms, nil
}

func (r *RedisRoomRepository) CreatePlayer(id int) (*Player, error) {
	username, errDB := r.userRepository.GetUserUsername(id)
	if errDB != nil {
		return nil, errDB
	}

	userId := strconv.Itoa(id)
	player := Player{
		ID:       userId,
		Username: username,
	}

	return &player, nil
}

func (r *RedisRoomRepository) AddPlayer(playerRequest *PlayerRequest) (*Room, error) {
	key := playerRequest.Room
	room, err := r.GetRoom(key)
	if err != nil {
		return nil, err
	}

	if room.Capacity == room.Players {
		return nil, apperrors.NewAppError(400, "Room is full", nil)
	}

	userId, err := strconv.Atoi(playerRequest.Player)
	if err != nil {
		return nil, apperrors.NewAppError(400, "Invalid player ID", err)
	}
	player, errDB := r.CreatePlayer(userId)
	if errDB != nil {
		return nil, apperrors.NewAppError(500, "Error creating player", errDB)
	}

	if len(room.Team1) <= len(room.Team2) {
		room.Team1 = append(room.Team1, *player)
	} else {
		room.Team2 = append(room.Team2, *player)
	}
	room.Players += 1

	if err := r.SaveRoom(room); err != nil {
		return nil, err
	}
	return room, nil
}

func (r *RedisRoomRepository) RemovePlayer(req *PlayerRequest) (*Room, error) {
	room, err := r.GetRoom(req.Room)
	if err != nil {
		return nil, err
	}

	if room.Players == 0 {
		return nil, apperrors.NewAppError(400, "Room is empty", errors.New("room is empty"))
	}

	players1 := room.Team1
	newPlayers := make([]Player, 0, len(players1))
	for _, p := range players1 {
		if p.ID != req.Player {
			newPlayers = append(newPlayers, p)
		}
	}
	room.Team1 = newPlayers

	players2 := room.Team2
	newPlayers2 := make([]Player, 0, len(players2))
	for _, p := range players2 {
		if p.ID != req.Player {
			newPlayers2 = append(newPlayers2, p)
		}
	}
	room.Team2 = newPlayers2
	room.Players -= 1
	if err := r.SaveRoom(room); err != nil {
		return nil, err
	}
	return room, nil
}

func (r *RedisRoomRepository) ChangeRoomOwner(roomId string, player Player) (*Room, error) {
	room, err := r.GetRoom(roomId)
	if err != nil {
		return nil, err
	}

	room.Host = player

	if err := r.SaveRoom(room); err != nil {
		return nil, apperrors.NewAppError(500, "Error saving room after changing owner", err)
	}

	return room, nil
}

func (r *RedisRoomRepository) DeleteRoom(id string) error {
	if err := r.db.Del(ctx, id).Err(); err != nil {
		log.Println("Error deleting room ", err)
		return apperrors.NewAppError(500, "Error deleting room", err)
	}
	if err := r.db.ZRem(ctx, "rooms_id", id).Err(); err != nil {
		log.Println("Error removing room id from list ", err)
		return apperrors.NewAppError(500, "Error removing room ID from list", err)
	}
	return nil
}

func (r *RedisRoomRepository) PublishToRoom(payload string) {
	if err := r.db.Publish(ctx, "messages", payload).Err(); err != nil {
		log.Println("Error publishing to room updates channel:", err)
	}
}
