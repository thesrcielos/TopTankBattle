package v1

import (
	"net/http"

	"github.com/thesrcielos/TopTankBattle/internal/user"

	"github.com/labstack/echo/v4"
)

func RegisterUserRoutes(g *echo.Group) {
	g.POST("/signup", SignupHandler)
	g.POST("/login", LoginHandler)
}

func SignupHandler(c echo.Context) error {
	var u user.User
	if err := c.Bind(&u); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	token, err := user.Signup(u)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, echo.Map{"token": token})
}

func LoginHandler(c echo.Context) error {
	var u user.User
	if err := c.Bind(&u); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	token, err := user.Login(u)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}
	return c.JSON(http.StatusOK, echo.Map{"token": token})
}
