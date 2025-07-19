package user

type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Username string `gorm:"uniqueIndex;not null" json:"username"`
	Password string `json:"password,omitempty"`
}

type UserStats struct {
	ID          uint `gorm:"primaryKey" json:"id"`
	UserID      uint `gorm:"not null" json:"user_id"`
	TotalGames  int  `json:"total_games"`
	TotalWins   int  `json:"total_wins"`
	TotalLosses int  `json:"total_losses"`
}

type UserStatsResponse struct {
	Username   string  `json:"username"`
	TotalGames int     `json:"totalGames"`
	Wins       int     `json:"wins"`
	Losses     int     `json:"losses"`
	WinRate    float64 `json:"winRate"`
}
