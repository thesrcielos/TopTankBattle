package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"log"

	"github.com/google/uuid"
	"github.com/thesrcielos/TopTankBattle/internal/user"
	"github.com/thesrcielos/TopTankBattle/pkg/db"
)

var ctx = context.Background()

func saveRoomRequest(RoomRequest *RoomRequest) (*Room, error) {
	player, errDB := createPlayer(RoomRequest.Player)
	if errDB != nil {
		return nil, errDB
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

	data, err := json.Marshal(room)
	if err != nil {
		return nil, fmt.Errorf("error serializing room request: %w", err)
	}

	if err := db.Rdb.Set(ctx, key, data, 0).Err(); err != nil {
		return nil, fmt.Errorf("error saving room: %w", err)
	}

	if err := db.Rdb.LPush(ctx, "room_ids", key).Err(); err != nil {
		return nil, fmt.Errorf("error saving room id to list: %w", err)
	}

	return &room, nil
}

func getRoom(key string) (*Room, error) {
	val, err := db.Rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("error getting room: %w", err)
	}
	var room Room
	if err := json.Unmarshal([]byte(val), &room); err != nil {
		return nil, fmt.Errorf("error deserializing: %w", err)
	}

	return &room, nil
}

func getRooms(page, pageSize int) (*[]Room, error) {
	start := int64(page * pageSize)
	end := start + int64(pageSize) - 1

	roomIDs, err := db.Rdb.LRange(ctx, "room_ids", start, end).Result()
	if err != nil {
		return nil, fmt.Errorf("error getting room ids: %w", err)
	}

	rooms := []Room{}
	for _, id := range roomIDs {
		room, err := getRoom(id)
		if err != nil {
			return nil, fmt.Errorf("error getting room with id %s: %w", id, err)
		}
		rooms = append(rooms, *room)
	}

	fmt.Println(rooms)
	return &rooms, nil
}

func createPlayer(id string) (*Player, error) {
	username, errDB := user.GetUserUsername(id)
	if errDB != nil {
		return nil, errDB
	}

	player := Player{
		ID:       id,
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
		return nil, errors.New("Room is full")
	}

	player, errDB := createPlayer(playerRequest.Player)
	if errDB != nil {
		return nil, errDB
	}

	if len(room.Team1) <= len(room.Team2) {
		room.Team1 = append(room.Team1, *player)
	} else {
		room.Team2 = append(room.Team2, *player)
	}
	room.Players += 1
	return room, nil
}

func removePlayer(req *PlayerRequest) (*Room, error) {
	room, err := getRoom(req.Room)
	if err != nil {
		return nil, err
	}
	if room.Capacity == 0 {
		return nil, errors.New("Room has no players left")
	}
	players1 := room.Team1
	newPlayers := make([]Player, 0, len(players1))
	for _, p := range players1 {
		if p.ID != req.Player {
			newPlayers = append(newPlayers, p)
		}
	}
	room.Team1 = newPlayers

	players2 := room.Team1
	newPlayers2 := make([]Player, 0, len(players2))
	for _, p := range players2 {
		if p.ID != req.Player {
			newPlayers2 = append(newPlayers2, p)
		}
	}
	room.Team2 = newPlayers2

	room.Players -= 1
	return room, nil
}

func deleteRoom(id string) error {
	if err := db.Rdb.Del(ctx, id).Err(); err != nil {
		log.Println("Error deleting room ", err)
		return err
	}

	return nil
}
