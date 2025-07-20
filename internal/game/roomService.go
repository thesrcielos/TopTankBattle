package game

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/thesrcielos/TopTankBattle/internal/apperrors"
	"github.com/thesrcielos/TopTankBattle/internal/game/state"
)

type RoomService struct {
	repo RoomRepository
}

func NewRoomService(repo RoomRepository) *RoomService {
	return &RoomService{repo: repo}
}

func (r *RoomService) CreateRoom(request *RoomRequest) (*Room, error) {
	val, errDB := r.repo.GetPlayerRoom(strconv.Itoa(request.Player))
	if errDB != nil {
		return nil, errDB
	}

	if val != nil {
		return nil, apperrors.NewAppError(400, "Player already in a room", nil)
	}

	room, err := r.repo.SaveRoomRequest(request)
	if err != nil {
		return nil, err
	}

	if err := r.repo.SavePlayerRoom(&PlayerRequest{
		Player: room.Host.ID,
		Room:   room.ID,
	}); err != nil {
		return nil, err
	}

	return room, nil
}

func (r *RoomService) GetRooms(request *RoomPageRequest) (*[]Room, error) {
	rooms, err := r.repo.GetRooms(request.Page, request.PageSize)
	if err != nil {
		return nil, err
	}

	return rooms, nil
}

func (r *RoomService) JoinRoom(playerRequest *PlayerRequest) (*Room, error) {
	val, errDB := r.repo.GetPlayerRoom(playerRequest.Player)
	if errDB != nil {
		return nil, errDB
	}
	if val != nil {
		return nil, apperrors.NewAppError(400, "Player already in a room", nil)
	}
	room, err := r.repo.AddPlayer(playerRequest)
	if err != nil {
		return nil, err
	}

	if err := r.repo.SavePlayerRoom(playerRequest); err != nil {
		return nil, err
	}

	if err := r.notifyPlayerJoin(room, playerRequest.Player); err != nil {
		return nil, err
	}

	return room, nil
}

func (r *RoomService) LeaveRoom(playerId string) error {
	val, errDB := r.repo.GetPlayerRoom(playerId)
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

	room, err := r.repo.RemovePlayer(playerRequest)
	if err != nil {
		return err
	}

	errDb := r.repo.DeletePlayerRoom(playerRequest.Player)
	if errDb != nil {
		return errDb
	}

	if err := r.changeOwnerIfNeeded(playerId, room); err != nil {
		return err
	}

	if err := r.notifyOrDeleteRoom(room, playerId); err != nil {
		return err
	}
	state.UnregisterPlayer(playerId)
	return nil
}

func (r *RoomService) KickPlayerFromRoom(playerId string, roomId string, playerToKick string) (*Room, error) {
	room, err := r.repo.GetRoom(roomId)
	if err != nil {
		return nil, err
	}

	playerRoom, errPR := r.repo.GetPlayerRoom(playerToKick)
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

	room, err = r.repo.RemovePlayer(playerRequest)
	if err != nil {
		return nil, err
	}

	errDb := r.repo.DeletePlayerRoom(playerToKick)
	if errDb != nil {
		return nil, errDb
	}

	if err := r.notifyPlayerKick(room, playerToKick); err != nil {
		return nil, err
	}

	state.UnregisterPlayer(playerToKick)

	return room, nil
}

func (r *RoomService) notifyPlayerKick(room *Room, kickedPlayerId string) error {
	message := GameMessage{
		Type: "ROOM_KICK",
		Payload: KickPlayerMessage{
			Room:   room.ID,
			Kicked: kickedPlayerId,
		},
	}
	room.Team1 = append(room.Team1, Player{ID: kickedPlayerId})
	r.sendRoomChangeMessage(room, message)
	return nil
}

func (r *RoomService) notifyPlayerJoin(room *Room, playerId string) error {
	player, err := r.getPlayerFromRoom(playerId, room)
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

	r.sendRoomChangeMessage(room, message)
	return nil
}

func (r *RoomService) sendRoomChangeMessage(room *Room, message GameMessage) {
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
	r.repo.PublishToRoom(string(msg))
}

func (r *RoomService) getPlayerFromRoom(playerId string, room *Room) (map[string]interface{}, error) {
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

func (r *RoomService) notifyOrDeleteRoom(room *Room, playerId string) error {
	if room.Players == 0 {
		return r.repo.DeleteRoom(room.ID)
	}

	message := GameMessage{
		Type: "ROOM_LEAVE",
		Payload: LeaveRoomMessage{
			Player: playerId,
			Host:   room.Host,
		},
	}

	r.sendRoomChangeMessage(room, message)
	return nil
}

func (r *RoomService) changeOwnerIfNeeded(playerID string, room *Room) error {
	if room.Host.ID != playerID || room.Players == 0 {
		return nil
	}
	var newHost Player
	if len(room.Team1) > 0 {
		newHost = room.Team1[0]
	} else {
		newHost = room.Team2[0]
	}
	_, err := r.repo.ChangeRoomOwner(room.ID, newHost)
	if err != nil {
		return err
	}
	room.Host = newHost
	return nil
}

func (r *RoomService) DeleteRoom(playerId string, roomId string) (*[]Player, error) {
	room, err := r.repo.GetRoom(roomId)
	if err != nil {
		return nil, err
	}

	if room.Host.ID != playerId {
		return nil, apperrors.NewAppError(403, "Only the host can delete the room", nil)
	}

	errDb := r.repo.DeleteRoom(roomId)
	if errDb != nil {
		return nil, errDb
	}

	players := []Player{}
	for _, player := range room.Team1 {
		players = append(players, player)
		r.repo.DeletePlayerRoom(player.ID)
	}

	for _, player := range room.Team2 {
		players = append(players, player)
		r.repo.DeletePlayerRoom(player.ID)
	}

	return &players, nil
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
