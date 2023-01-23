package store

import (
	"github.com/julez-dev/neveralone/internal/party"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type Session struct {
	m cmap.ConcurrentMap[string, *party.Session]
}

func NewSession() *Session {
	m := cmap.New[*party.Session]()
	return &Session{m: m}
}

func (s *Session) Get(id string) (*party.Session, bool) {
	return s.m.Get(id)
}

func (s *Session) Set(session *party.Session) {
	s.m.Set(session.ID.String(), session)
}

func (s *Session) Delete(id string) {
	s.m.Remove(id)
}

//func (s *Session) Close() error {
//	sessions := s.m.Items()
//
//	for _, session := range sessions {
//		session.Close()
//	}
//
//	s.m.Clear()
//
//	return nil
//}
