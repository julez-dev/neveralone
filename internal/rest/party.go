package rest

import (
	"crypto/subtle"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/julez-dev/neveralone/internal/metric"
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
	ServeParty(io.Writer, string, *party.User) error
	ServeCreateParty(io.Writer, *party.User) error
	ServeJoinPartyPassword(io.Writer, *party.User, string, bool) error
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

	params, err := c.FormParams()

	if err != nil {
		return err
	}

	config := &party.Config{
		Visibility: party.PrivateLobby,
	}

	if params.Has("is_custom") {
		if visibility := params.Get("visibility"); visibility == "public" {
			config.Visibility = party.PublicLobby
		}

		if passwordProtected := params.Get("passphrase"); passwordProtected == "yes_passphrase" {
			config.HasPassphrase = true
		}

		config.Passphrase = params.Get("passphrase-lobby")

		if allowOnlyHost := params.Get("only_host"); allowOnlyHost == "yes_only_host" {
			config.AllowOnlyHost = true
		}
	}

	session := party.NewSession(s.logger, user, config)
	s.sessionStore.Set(session)

	go func() {
		if config.Visibility == party.PrivateLobby {
			metric.RoomCountPrivate.Inc()
			defer metric.RoomCountPrivate.Dec()
		} else {
			metric.RoomCountPublic.Inc()
			defer metric.RoomCountPublic.Dec()
		}

		session.Run(s.closeCTX)
		s.sessionStore.Delete(session.ID.String())
	}()

	url := fmt.Sprintf("/party/%s", session.ID.String())
	return c.Redirect(http.StatusMovedPermanently, url)
}

func (s *Server) CreatePartyCustom(c echo.Context) error {
	user, ok := c.Request().Context().Value(userKey).(*party.User)

	if !ok {
		return fmt.Errorf("could not parse user from context")
	}

	return s.partyHandler.ServeCreateParty(c.Response(), user)
}

func (s *Server) GetParty(c echo.Context) error {
	user, ok := c.Request().Context().Value(userKey).(*party.User)

	if !ok {
		return fmt.Errorf("could not parse user from context")
	}

	sessionID := c.Param("id")

	session, ok := s.sessionStore.Get(sessionID)

	if !ok {
		return c.Redirect(http.StatusMovedPermanently, "/?not_found=1")
	}

	cfg := session.GetConfig()

	// session has a password and user was not already connected
	if cfg.HasPassphrase && !session.HasPlayerIDInLobby(user.ID.String()) {
		uPass := c.FormValue("passphrase-lobby")

		if uPass == "" {
			return s.partyHandler.ServeJoinPartyPassword(c.Response(), user, session.ID.String(), false)
		}

		if subtle.ConstantTimeCompare([]byte(uPass), []byte(cfg.Passphrase)) != 1 {
			return s.partyHandler.ServeJoinPartyPassword(c.Response(), user, session.ID.String(), true)
		}
	}

	session.Register <- user

	return s.partyHandler.ServeParty(c.Response(), sessionID, user)
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

	if !session.HasPlayerIDInLobby(user.ID.String()) {
		return c.NoContent(http.StatusUnauthorized)
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
