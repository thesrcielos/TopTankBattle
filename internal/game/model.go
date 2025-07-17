package game

type Player struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type PlayerRequest struct {
	Player string `json:"player"`
	Room   string `json:"room"`
}

type RoomRequest struct {
	Name     string `json:"name"`
	Player   int    `json:"player"`
	Capacity int    `json:"capacity"`
}

type Room struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Capacity int      `json:"capacity"`
	Players  int      `json:"players"`
	Team1    []Player `json:"team1"`
	Team2    []Player `json:"team2"`
	Host     Player   `json:"host"`
	Status   string   `json:"status"`
}

type RoomPageRequest struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

type JoinRoomMessage struct {
	Player Player `json:"player"`
	Team   int    `json:"team"`
}

type LeaveRoomMessage struct {
	Player string `json:"player"`
	Host   Player `json:"host"`
}

type KickPlayerMessage struct {
	Room   string `json:"roomId"`
	Kicked string `json:"kicked"`
}

type MoveMessage struct {
	PlayerId string      `json:"playerId"`
	Position interface{} `json:"position"`
}

type ShootMessage struct {
	ID       string      `json:"id"`
	Position interface{} `json:"position"`
	Team1    bool        `json:"team1"`
	OwnerId  string      `json:"ownerId"`
}

type GameMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	Users   []string    `json:"users"`
}

type GameStateMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
