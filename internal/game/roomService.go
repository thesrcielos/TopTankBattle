package game

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/thesrcielos/TopTankBattle/internal/apperrors"
)

func CreateRoom(request *RoomRequest) (*Room, error) {
	val, errDB := getPlayerRoom(strconv.Itoa(request.Player))
	if errDB != nil {
		return nil, errDB
	}

	if val != nil {
		return nil, apperrors.NewAppError(400, "Player already in a room", nil)
	}

	room, err := saveRoomRequest(request)
	if err != nil {
		return nil, err
	}

	if err := savePlayerRoom(&PlayerRequest{
		Player: room.Host.ID,
		Room:   room.ID,
	}); err != nil {
		return nil, err
	}

	SubscribeToRoom(room.ID)
	return room, nil
}

func GetRooms(request *RoomPageRequest) (*[]Room, error) {
	rooms, err := getRooms(request.Page, request.PageSize)
	if err != nil {
		return nil, err
	}

	return rooms, nil
}

func JoinRoom(playerRequest *PlayerRequest) (*Room, error) {
	val, errDB := getPlayerRoom(playerRequest.Player)
	if errDB != nil {
		return nil, errDB
	}
	if val != nil {
		return nil, apperrors.NewAppError(400, "Player already in a room", nil)
	}
	room, err := addPlayer(playerRequest)
	if err != nil {
		return nil, err
	}

	if err := savePlayerRoom(playerRequest); err != nil {
		return nil, err
	}

	if err := notifyPlayerJoin(room, playerRequest.Player); err != nil {
		return nil, err
	}

	return room, nil
}

func LeaveRoom(playerId string) error {
	val, errDB := getPlayerRoom(playerId)
	if errDB != nil {
		return errDB
	}
	if val == nil {
		return apperrors.NewAppError(400, "Player is not in a room", nil)
	}
	playerRequest := &PlayerRequest{
		Player: playerId,
		Room:   val.(string),
	}

	room, err := removePlayer(playerRequest)
	if err != nil {
		return err
	}

	errDb := deletePlayerRoom(playerRequest.Player)
	if errDb != nil {
		return errDb
	}

	if err := changeOwnerIfNeeded(playerId, room); err != nil {
		return err
	}

	if err := notifyOrDeleteRoom(room, playerId); err != nil {
		return err
	}

	return nil
}

func KickPlayerFromRoom(playerId string, roomId string, playerToKick string) (*Room, error) {
	room, err := getRoom(roomId)
	if err != nil {
		return nil, err
	}

	playerRoom, errPR := getPlayerRoom(playerToKick)
	if errPR != nil {
		return nil, errPR
	}

	if playerRoom == nil || playerRoom.(string) != roomId {
		return nil, apperrors.NewAppError(404, "Player not found in room", nil)
	}

	if room.Host.ID != playerId {
		return nil, apperrors.NewAppError(403, "Only the host can kick players", nil)
	}

	if playerToKick == room.Host.ID {
		return nil, apperrors.NewAppError(403, "Host cannot be kicked", nil)
	}

	playerRequest := &PlayerRequest{
		Player: playerToKick,
		Room:   roomId,
	}

	room, err = removePlayer(playerRequest)
	if err != nil {
		return nil, err
	}

	errDb := deletePlayerRoom(playerToKick)
	if errDb != nil {
		return nil, errDb
	}

	if err := notifyPlayerKick(room, playerToKick); err != nil {
		return nil, err
	}

	return room, nil
}

func notifyPlayerKick(room *Room, kickedPlayerId string) error {
	message := GameMessage{
		Type: "ROOM_KICK",
		Payload: KickPlayerMessage{
			Room:   room.ID,
			Kicked: kickedPlayerId,
		},
	}
	room.Team1 = append(room.Team1, Player{ID: kickedPlayerId})
	go sendRoomChangeMessage(room, message)
	return nil
}

func notifyPlayerJoin(room *Room, playerId string) error {
	player, err := getPlayerFromRoom(playerId, room)
	if err != nil {
		return err
	}

	message := GameMessage{
		Type: "ROOM_JOIN",
		Payload: JoinRoomMessage{
			Player: player["player"].(Player),
			Team:   player["team"].(int),
		},
	}

	go sendRoomChangeMessage(room, message)
	return nil
}

func sendRoomChangeMessage(room *Room, message GameMessage) {
	players := make([]string, 0, len(room.Team1)+len(room.Team2))
	for _, player := range room.Team1 {
		players = append(players, player.ID)
	}
	for _, player := range room.Team2 {
		players = append(players, player.ID)
	}
	message.Users = players
	msg, err := json.Marshal(message)
	if err != nil {
		log.Println("Error encoding message:", err)
		return
	}
	PublishToRoom(room.ID, string(msg))
}

func getPlayerFromRoom(playerId string, room *Room) (map[string]interface{}, error) {
	for _, player := range room.Team1 {
		if player.ID == playerId {
			return map[string]interface{}{
				"player": player,
				"team":   1,
			}, nil
		}
	}
	for _, player := range room.Team2 {
		if player.ID == playerId {
			return map[string]interface{}{
				"player": player,
				"team":   2,
			}, nil
		}
	}
	return nil, apperrors.NewAppError(404, "Player not found in room", nil)
}

func notifyOrDeleteRoom(room *Room, playerId string) error {
	if room.Players == 0 {
		if err := UnsubscribeFromRoom(room.ID); err != nil {
			return err
		}
		return deleteRoom(room.ID)
	}

	message := GameMessage{
		Type: "ROOM_LEAVE",
		Payload: LeaveRoomMessage{
			Player: playerId,
			Host:   room.Host,
		},
	}

	go sendRoomChangeMessage(room, message)
	return nil
}

func changeOwnerIfNeeded(playerID string, room *Room) error {
	if room.Host.ID != playerID || room.Players == 0 {
		return nil
	}
	var newHost Player
	if len(room.Team1) > 0 {
		newHost = room.Team1[0]
	} else {
		newHost = room.Team2[0]
	}
	_, err := changeRoomOwner(room.ID, newHost)
	if err != nil {
		return err
	}
	room.Host = newHost
	return nil
}

func DeleteRoom(playerId string, roomId string) (*[]Player, error) {
	room, err := getRoom(roomId)
	if err != nil {
		return nil, err
	}

	if room.Host.ID != playerId {
		return nil, apperrors.NewAppError(403, "Only the host can delete the room", nil)
	}

	errDb := deleteRoom(roomId)
	if errDb != nil {
		return nil, errDb
	}

	players := []Player{}
	for _, player := range room.Team1 {
		players = append(players, player)
		deletePlayerRoom(player.ID)
	}

	for _, player := range room.Team2 {
		players = append(players, player)
		deletePlayerRoom(player.ID)
	}

	return &players, nil
}

func findRoom(key string) (*Room, error) {
	if len(key) != 8 {
		return nil, apperrors.NewAppError(400, "Room id must be of 8 characters", nil)
	}

	room, err := getRoom(key)
	if err != nil {
		return nil, err
	}

	return room, nil
}

func (r *RoomRequest) Validate() error {
	validCapacities := map[int]bool{2: true, 4: true, 6: true}
	if !validCapacities[r.Capacity] {
		return apperrors.NewAppError(400, "capacity must be 2, 4, or 6", nil)
	}

	if len(r.Name) > 30 {
		return apperrors.NewAppError(400, "name must not exceed 30 characters", nil)
	}

	return nil
}
