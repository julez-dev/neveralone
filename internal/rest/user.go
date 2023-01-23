package rest

import (
	"context"
	"errors"
	"github.com/julez-dev/neveralone/internal/party"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

type contextKey int

const (
	userKey contextKey = iota
)

type userStore interface {
	Get(string) (*party.User, bool)
	Set(*party.User)
}

func (s *Server) getUserMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie("anon_user")

		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				user := party.NewRandomUser()
				setUserCookie(c, user)
				s.userStore.Set(user)

				ctx := context.WithValue(c.Request().Context(), userKey, user)
				c.SetRequest(c.Request().WithContext(ctx))

				return next(c)
			}

			return err
		}

		user, ok := s.userStore.Get(cookie.Value)

		if !ok {
			user = party.NewRandomUser()
			setUserCookie(c, user)
			s.userStore.Set(user)
		}

		ctx := context.WithValue(c.Request().Context(), userKey, user)
		c.SetRequest(c.Request().WithContext(ctx))

		return next(c)
	}
}

func setUserCookie(c echo.Context, user *party.User) {
	c.SetCookie(&http.Cookie{
		Name:   "anon_user",
		Value:  user.ID.String(),
		Path:   "/",
		MaxAge: int(time.Hour * 24 * 30),
		Secure: c.Request().TLS != nil,
	})
}
