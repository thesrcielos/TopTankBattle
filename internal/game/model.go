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
	Player   string `json:"player"`
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
	Status   string   `json:"state"`
}

type RoomPageRequest struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}
