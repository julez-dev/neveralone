package rest

import (
	"github.com/labstack/echo/v4"
	"io"
)

type homeHandler interface {
	ServeHome(writer io.Writer) error
}

func (s *Server) GetHome(ctx echo.Context) error {
	return s.homeHandler.ServeHome(ctx.Response())
}
