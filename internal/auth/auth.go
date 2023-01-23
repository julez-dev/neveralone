package auth

import (
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/julez-dev/neveralone/internal/party"
	"time"
)

type JWT struct {
	secret []byte
}

func NewJWT(secret []byte) *JWT {
	return &JWT{secret: secret}
}

func (j *JWT) GenerateToken(user *party.User, expiresAt time.Time) (string, error) {
	type Claims struct {
		Name string `json:"name"`
		jwt.RegisteredClaims
	}

	claims := Claims{
		Name: user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "neveralone.tv",
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

func (j *JWT) ValidateToken(signedToken string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(signedToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return j.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("could not parse token")
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("token is not valid")
}
