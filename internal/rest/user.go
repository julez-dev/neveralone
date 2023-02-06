package rest

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/julez-dev/neveralone/internal/metric"
	"github.com/julez-dev/neveralone/internal/party"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

type contextKey int

const (
	userKey contextKey = iota
)

const (
	tokenDuration = time.Hour * 24 * 30
	cookieName    = "user"
)

type jwtGenerator interface {
	GenerateToken(*party.User, time.Time) (string, error)
	ValidateToken(string) (jwt.MapClaims, error)
}

func (s *Server) getUserMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie(cookieName)

		// User does not have a cookie yet
		if err != nil {
			if !errors.Is(err, http.ErrNoCookie) {
				return err
			}

			if err := generateAndSetNewUser(c, s.jwtGenerator); err != nil {
				return err
			}

			return next(c)
		}

		claims, err := s.jwtGenerator.ValidateToken(cookie.Value)

		// Just set new user if not valid anymore
		if err != nil {
			s.logger.Info().Str("old-token", cookie.Value).Err(err).Msg("creating new user")
			if err := generateAndSetNewUser(c, s.jwtGenerator); err != nil {
				return err
			}

			return next(c)
		}

		user := &party.User{
			ID:   uuid.Must(uuid.Parse(claims["sub"].(string))),
			Name: claims["name"].(string),
		}

		ctx := context.WithValue(c.Request().Context(), userKey, user)
		c.SetRequest(c.Request().WithContext(ctx))

		//if refreshedToken, err := s.jwtGenerator.GenerateToken(user, time.Now().Add(tokenDuration)); err != nil {
		//	setUserCookie(c, refreshedToken)
		//}

		return next(c)
	}
}

func generateAndSetNewUser(c echo.Context, jwt jwtGenerator) error {
	user := party.NewRandomUser()
	token, err := jwt.GenerateToken(user, time.Now().Add(tokenDuration))

	if err != nil {
		return err
	}

	metric.NewUsersGenerated.Inc()

	setUserCookie(c, token)
	ctx := context.WithValue(c.Request().Context(), userKey, user)
	c.SetRequest(c.Request().WithContext(ctx))
	return nil
}

func setUserCookie(c echo.Context, token string) {
	c.SetCookie(&http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(tokenDuration),
		Secure:   c.Request().TLS != nil,
	})
}
