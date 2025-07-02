package game

import (
	"errors"
)

func CreateRoom(request *RoomRequest) (*Room, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	room, err := saveRoomRequest(request)
	if err != nil {
		return nil, err
	}

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
	room, err := addPlayer(playerRequest)
	if err != nil {
		return nil, err
	}

	//go actions.HandleRoomJoin(playerRequest.Player, room)
	return room, nil
}

func LeaveRoom(playerRequest *PlayerRequest) (*Room, error) {
	room, err := removePlayer(playerRequest)
	if err != nil {
		return nil, err
	}

	return room, nil
}

func DeleteRoom(playerId string, roomId string) (*[]Player, error) {
	room, err := getRoom(roomId)
	if err != nil {
		return nil, err
	}

	if room.Host.ID != playerId {
		return nil, errors.New("Only the host can delete the room")
	}

	errDb := deleteRoom(roomId)
	if errDb != nil {
		return nil, errDb
	}

	players := []Player{}
	for _, player := range room.Team1 {
		players = append(players, player)
	}

	for _, player := range room.Team2 {
		players = append(players, player)
	}

	return &players, nil
}

func findRoom(key string) (*Room, error) {
	if len(key) != 8 {
		return nil, errors.New("Room id must be of 8 characters")
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
		return errors.New("capacity must be 2, 4, or 6")
	}

	if len(r.Name) > 30 {
		return errors.New("name must not exceed 30 characters")
	}

	return nil
}
