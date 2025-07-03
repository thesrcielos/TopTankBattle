package v1

import (
	"net/http"

	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/thesrcielos/TopTankBattle/internal/game"
)

func RegisterRoomRoutes(g *echo.Group) {
	g.POST("", CreateRoomHandler)
	g.GET("", GetRoomsHandler)
	g.POST("/players", JoinRoomHandler)
	g.DELETE("/players", LeaveRoomHandler)
}

func CreateRoomHandler(c echo.Context) error {
	var r game.RoomRequest
	if err := c.Bind(&r); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	room, err := game.CreateRoom(&r)

	if err != nil {
		fmt.Println("Error creating room:", err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, echo.Map{
		"room": room,
	})
}

func GetRoomsHandler(c echo.Context) error {
	var p game.RoomPageRequest
	if err := c.Bind(&p); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	rooms, err := game.GetRooms(&p)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, echo.Map{
		"rooms": rooms,
	})
}

func JoinRoomHandler(c echo.Context) error {
	var p game.PlayerRequest
	if err := c.Bind(&p); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	room, err := game.JoinRoom(&p)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusAccepted, echo.Map{
		"room": room,
	})
}

func LeaveRoomHandler(c echo.Context) error {
	var p game.PlayerRequest
	if err := c.Bind(&p); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	room, err := game.LeaveRoom(&p)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusAccepted, echo.Map{
		"room": room,
	})
}
