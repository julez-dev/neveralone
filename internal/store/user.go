package store

import (
	"github.com/julez-dev/neveralone/internal/party"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type User struct {
	m cmap.ConcurrentMap[string, *party.User]
}

func NewUser() *User {
	m := cmap.New[*party.User]()
	return &User{m: m}
}

func (u *User) Get(id string) (*party.User, bool) {
	return u.m.Get(id)
}

func (u *User) Set(pu *party.User) {
	u.m.Set(pu.ID.String(), pu)
}
