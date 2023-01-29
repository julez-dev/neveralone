package rest

import (
	"fmt"
	"github.com/julez-dev/neveralone/internal/party"
	"github.com/labstack/echo/v4"
	"io"
)

type homeHandler interface {
	ServeHome(io.Writer, *party.User) error
}

func (s *Server) GetHome(c echo.Context) error {
	user, ok := c.Request().Context().Value(userKey).(*party.User)

	if !ok {
		return fmt.Errorf("could not parse user from context")
	}

	return s.homeHandler.ServeHome(c.Response(), user)
}
