package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	api_middleware "github.com/thesrcielos/TopTankBattle/api/middleware"
	v1 "github.com/thesrcielos/TopTankBattle/api/v1"
	"github.com/thesrcielos/TopTankBattle/internal/apperrors"
	"github.com/thesrcielos/TopTankBattle/internal/game"
	"github.com/thesrcielos/TopTankBattle/internal/game/maps"
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
	db.DB.AutoMigrate(&user.UserStats{})
	maps.GenerateCollisionMatrix("map.json")
	inyectDependencies()
	e := echo.New()

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		var appErr *apperrors.AppError

		if errors.As(err, &appErr) {
			_ = c.JSON(appErr.Code, map[string]string{
				"error": appErr.Message,
			})
		} else if he, ok := err.(*echo.HTTPError); ok {
			_ = c.JSON(he.Code, map[string]string{
				"error": fmt.Sprintf("%v", he.Message),
			})
		} else {
			_ = c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Internal Server Error",
			})
		}
	}

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	api := e.Group("/api/v1")
	v1.RegisterUserRoutes(api.Group("/users"))

	g := api.Group("/rooms")
	g.Use(api_middleware.SetupJWTMiddleware())
	v1.RegisterRoomRoutes(g)

	e.GET("/game", websocket.WebSocketHandler)
	e.GET("/deleteAll", func(c echo.Context) error {
		if err := db.Rdb.FlushAll(context.Background()).Err(); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete all data")
		}
		return c.JSON(http.StatusOK, echo.Map{"message": "All data deleted successfully"})
	})
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, echo.Map{"ok": true})
	})
	e.Logger.Fatal(e.Start(":8080"))

}

func inyectDependencies() {
	var gameServiceImp *game.GameServiceImpl
	redisRepository := game.NewGameStateRepository(gameServiceImp, db.Rdb)
	userRepository := user.NewUserRepository(db.DB)
	roomRepository := game.NewRedisRoomRepository(userRepository, db.Rdb)
	roomService := game.NewRoomService(roomRepository)
	userService := user.NewUserService(userRepository)
	gameServiceImp = game.NewGameService(redisRepository, roomRepository, roomService, userService)
	redisRepository.SetLeaderElector(gameServiceImp)
	v1.RoomService = roomService
	v1.UserService = userService
	websocket.RoomService = roomService
	websocket.GameService = gameServiceImp

	startRedisSubscriber(redisRepository)
}

func startRedisSubscriber(repo game.GameStateRepository) {
	go func() {
		for {
			log.Println("Intentando suscribirse al canal Redis...")
			err := repo.SubscribeMessages()
			if err != nil {
				log.Printf("Fallo al suscribirse: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			log.Println("¡Suscripción exitosa!")
			return
		}
	}()
}
