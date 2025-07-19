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
	"github.com/thesrcielos/TopTankBattle/pkg/db"
)

var ctx = context.Background()

func saveRoomRequest(RoomRequest *RoomRequest) (*Room, error) {
	player, errDB := createPlayer(RoomRequest.Player)
	if errDB != nil {
		return nil, apperrors.NewAppError(500, "Error creating player", errDB)
	}

	key := uuid.New().String()[:8]
	room := Room{
		ID:       key,
		Name:     RoomRequest.Name,
		Capacity: RoomRequest.Capacity,
		Players:  1,
		Team1:    []Player{*player},
		Team2:    []Player{},
		Host:     *player,
		Status:   "LOBBY",
	}

	if err := saveRoom(room); err != nil {
		return nil, err
	}

	timestamp := float64(time.Now().Unix())
	if err := db.Rdb.ZAdd(ctx, "rooms_id", redis.Z{Score: timestamp, Member: room.ID}).Err(); err != nil {
		return nil, apperrors.NewAppError(500, "Error saving room ID", err)
	}

	return &room, nil
}

func saveRoom(room Room) error {
	data, err := json.Marshal(room)
	if err != nil {
		return apperrors.NewAppError(500, "Error serializing room data", err)
	}

	if err := db.Rdb.Set(ctx, room.ID, data, 0).Err(); err != nil {
		return apperrors.NewAppError(500, "Error saving room", err)
	}

	return nil
}

func savePlayerRoom(playerRequest *PlayerRequest) error {
	if err := db.Rdb.Set(ctx, playerRequest.Player, playerRequest.Room, 0).Err(); err != nil {
		fmt.Println("Error saving player room:", err)
		return apperrors.NewAppError(500, "Error saving player room", err)
	}

	return nil
}

func getPlayerRoom(playerId string) (interface{}, error) {
	val, err := db.Rdb.Get(ctx, playerId).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, apperrors.NewAppError(500, "Error getting player room", err)
	}

	return val, nil
}

func deletePlayerRoom(playerId string) error {
	if err := db.Rdb.Del(ctx, playerId).Err(); err != nil {
		return apperrors.NewAppError(500, "Error deleting player room", err)
	}

	return nil
}

func getRoom(key string) (*Room, error) {
	val, err := db.Rdb.Get(ctx, key).Result()
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

func getRooms(page, pageSize int) (*[]Room, error) {
	start := int64(page * pageSize)
	end := start + int64(pageSize) - 1

	roomIDs, err := db.Rdb.ZRevRange(ctx, "rooms_id", start, end).Result()
	if err != nil {
		return nil, apperrors.NewAppError(500, "Error getting room IDs", err)
	}

	rooms := []Room{}
	for _, id := range roomIDs {
		room, err := getRoom(id)
		if err != nil {
			return nil, apperrors.NewAppError(500, "Error getting room by ID", err)
		}
		rooms = append(rooms, *room)
	}

	return &rooms, nil
}

func createPlayer(id int) (*Player, error) {
	username, errDB := user.GetUserUsername(id)
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

func addPlayer(playerRequest *PlayerRequest) (*Room, error) {
	key := playerRequest.Room
	room, err := getRoom(key)
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
	player, errDB := createPlayer(userId)
	if errDB != nil {
		return nil, apperrors.NewAppError(500, "Error creating player", errDB)
	}

	if len(room.Team1) <= len(room.Team2) {
		room.Team1 = append(room.Team1, *player)
	} else {
		room.Team2 = append(room.Team2, *player)
	}
	room.Players += 1

	if err := saveRoom(*room); err != nil {
		return nil, err
	}
	return room, nil
}

func removePlayer(req *PlayerRequest) (*Room, error) {
	room, err := getRoom(req.Room)
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
	if err := saveRoom(*room); err != nil {
		return nil, err
	}
	return room, nil
}

func changeRoomOwner(roomId string, player Player) (*Room, error) {
	room, err := getRoom(roomId)
	if err != nil {
		return nil, err
	}

	room.Host = player

	if err := saveRoom(*room); err != nil {
		return nil, apperrors.NewAppError(500, "Error saving room after changing owner", err)
	}

	return room, nil
}

func deleteRoom(id string) error {
	if err := db.Rdb.Del(ctx, id).Err(); err != nil {
		log.Println("Error deleting room ", err)
		return apperrors.NewAppError(500, "Error deleting room", err)
	}
	if err := db.Rdb.ZRem(ctx, "rooms_id", id).Err(); err != nil {
		log.Println("Error removing room id from list ", err)
		return apperrors.NewAppError(500, "Error removing room ID from list", err)
	}
	return nil
}
