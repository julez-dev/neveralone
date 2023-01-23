package rest

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/julez-dev/neveralone/internal/party"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type partyTemplateHandler interface {
	ServeParty(writer io.Writer, id string) error
}

type sessionStore interface {
	Get(string) (*party.Session, bool)
	Set(*party.Session)
	Delete(string)
}

func (s *Server) CreateParty(c echo.Context) error {
	user, ok := c.Request().Context().Value(userKey).(*party.User)

	if !ok {
		return fmt.Errorf("could not parse user from context")
	}

	session := party.NewSession(s.logger, user)
	s.sessionStore.Set(session)

	go func() {
		session.Run(s.closeCTX)
		s.sessionStore.Delete(session.ID.String())
	}()

	url := fmt.Sprintf("/party/%s", session.ID.String())
	return c.Redirect(http.StatusMovedPermanently, url)
}

func (s *Server) GetParty(c echo.Context) error {
	user, ok := c.Request().Context().Value(userKey).(*party.User)

	if !ok {
		return fmt.Errorf("could not parse user from context")
	}

	sessionID := c.Param("id")

	session, ok := s.sessionStore.Get(sessionID)

	if !ok {
		return c.NoContent(http.StatusNotFound)
	}

	session.Register <- user

	return s.partyHandler.ServeParty(c.Response(), sessionID)
}

func (s *Server) JoinWS(c echo.Context) error {
	user, ok := c.Request().Context().Value(userKey).(*party.User)

	if !ok {
		return fmt.Errorf("could not parse user from context")
	}

	sessionID := c.Param("id")
	session, ok := s.sessionStore.Get(sessionID)

	if !ok {
		return c.NoContent(http.StatusNotFound)
	}

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	session.AttachWS <- &party.AttachSocket{
		PlayerID:   user.ID.String(),
		Connection: ws,
	}

	return nil
}
