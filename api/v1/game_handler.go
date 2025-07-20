package v1

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/thesrcielos/TopTankBattle/internal/game"
)

const INVALID_REQUEST = "invalid request"

var RoomService *game.RoomService

func RegisterRoomRoutes(g *echo.Group) {
	g.POST("", CreateRoomHandler)
	g.GET("", GetRoomsHandler)
	g.POST("/players", JoinRoomHandler)
	g.DELETE("/players", LeaveRoomHandler)
}

func CreateRoomHandler(c echo.Context) error {
	var r game.RoomRequest
	if err := c.Bind(&r); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, INVALID_REQUEST)
	}

	room, err := RoomService.CreateRoom(&r)

	if err != nil {
		fmt.Println("Error creating room:", err)
		return err
	}
	return c.JSON(http.StatusCreated, echo.Map{
		"room": room,
	})
}

func GetRoomsHandler(c echo.Context) error {
	page := c.QueryParam("page")
	pageSize := c.QueryParam("size")
	if page == "" || pageSize == "" {
		return echo.NewHTTPError(http.StatusBadRequest, INVALID_REQUEST)
	}

	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, INVALID_REQUEST)
	}
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil || pageSizeInt <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, INVALID_REQUEST)
	}

	rooms, err := RoomService.GetRooms(&game.RoomPageRequest{
		Page:     pageInt,
		PageSize: pageInt})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{
		"rooms": rooms,
	})
}

func JoinRoomHandler(c echo.Context) error {
	var p game.PlayerRequest
	if err := c.Bind(&p); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, INVALID_REQUEST)
	}
	room, err := RoomService.JoinRoom(&p)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusAccepted, echo.Map{
		"room": room,
	})
}

func LeaveRoomHandler(c echo.Context) error {
	playerId := c.QueryParam("playerId")
	if playerId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "playerId is required")
	}
	err := RoomService.LeaveRoom(playerId)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusAccepted, echo.Map{
		"room": true,
	})
}
