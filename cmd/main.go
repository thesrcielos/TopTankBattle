package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	api_middleware "github.com/thesrcielos/TopTankBattle/api/middleware"
	v1 "github.com/thesrcielos/TopTankBattle/api/v1"
	"github.com/thesrcielos/TopTankBattle/internal/user"
	"github.com/thesrcielos/TopTankBattle/pkg/db"
	"github.com/thesrcielos/TopTankBattle/websocket"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️File .env not found, using system values")
	}

	db.Init()
	db.DB.AutoMigrate(&user.User{})

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	api := e.Group("/api/v1")
	v1.RegisterUserRoutes(api.Group("/users"))

	g := api.Group("/rooms")
	g.Use(api_middleware.SetupJWTMiddleware())
	v1.RegisterRoomRoutes(g)

	e.GET("/game", websocket.WebSocketHandler)

	e.Logger.Fatal(e.Start(":8080"))
}
