package party

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"sync"
	"time"
)

const (
	usageCheckWait = time.Second * 45
)

type Session struct {
	logger zerolog.Logger
	ID     uuid.UUID

	videoID string
	vLock   *sync.Mutex

	pLock       *sync.Mutex
	players     map[string]*Player
	connections map[string]*connection

	broadcast  chan *broadcastMessage
	Register   chan *User
	Unregister chan *User

	AttachWS chan *AttachSocket
	DetachWS chan string

	o *sync.Once
}

type AttachSocket struct {
	PlayerID   string
	Connection *websocket.Conn
}

type broadcastMessage struct {
	playerID string
	data     []byte
}

func NewSession(logger zerolog.Logger, host *User) *Session {
	id := uuid.New()

	s := &Session{
		logger:      logger.With().Str("room-id", id.String()).Logger(),
		ID:          id,
		players:     map[string]*Player{},
		connections: map[string]*connection{},
		broadcast:   make(chan *broadcastMessage),
		Register:    make(chan *User),
		Unregister:  make(chan *User),
		AttachWS:    make(chan *AttachSocket),
		DetachWS:    make(chan string),
		o:           &sync.Once{},
		pLock:       &sync.Mutex{},
		vLock:       &sync.Mutex{},
	}

	s.players[host.ID.String()] = NewPlayer(host, true)

	return s
}

func (s *Session) GetPlayersCopy() []*Player {
	s.pLock.Lock()
	defer s.pLock.Unlock()

	copied := make([]*Player, 0, len(s.players))

	for _, p := range s.players {
		copied = append(copied, &Player{
			User: &User{
				ID:   p.User.ID,
				Name: p.User.Name,
			},
			IsHost: p.IsHost,
		})
	}

	return copied
}

func (s *Session) GetCurrentVideoID() string {
	s.vLock.Lock()
	defer s.vLock.Unlock()
	return s.videoID
}

func (s *Session) Run(ctx context.Context) {
	ticker := time.NewTicker(usageCheckWait)

	defer func() {
		ticker.Stop()
		s.logger.Debug().Msg("session loop exited")
	}()

	for {
		select {
		case <-ticker.C:
			// Close room when no more connections
			if len(s.connections) < 1 {
				return
			}
		case user := <-s.Register:
			s.pLock.Lock()
			id := user.ID.String()
			_, exists := s.players[id]

			if !exists {
				s.players[id] = NewPlayer(user, false)
			}
			s.pLock.Unlock()

			ticker.Reset(usageCheckWait)
		case user := <-s.Unregister:
			s.pLock.Lock()
			id := user.ID.String()
			_, ok := s.players[id]

			if ok {
				delete(s.players, id)
			}
			s.pLock.Unlock()

			ticker.Reset(usageCheckWait)
		case ws := <-s.AttachWS:
			_, exists := s.connections[ws.PlayerID]

			if exists {
				s.logger.Info().Str("user-id", ws.PlayerID).Msg("user with active ws connection tried to join")

				ws.Connection.SetWriteDeadline(time.Now().Add(writeWait))
				err := ws.Connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, "user already has active connection for this room"))

				if err != nil {
					s.logger.Err(err).Str("user-id", ws.PlayerID).Msg("error closing blocked ws")
				}

				break
			}

			newConn := newConnection(s.logger, ws.PlayerID, s, ws.Connection)
			s.connections[ws.PlayerID] = newConn

			// start reader
			go func(conn *connection) {
				conn.readWS()
			}(newConn)

			// start writer
			go func(conn *connection) {
				conn.writeWS()
			}(newConn)

			ticker.Reset(usageCheckWait)
		case id := <-s.DetachWS:
			if conn, ok := s.connections[id]; ok {
				delete(s.connections, id)
				conn.closeSendOnce()
			}

			ticker.Reset(usageCheckWait)
		case event := <-s.broadcast:
			s.hookBroadcast(event)

			for _, conn := range s.connections {
				if conn.userID != event.playerID {
					s.logger.Debug().
						Str("sender-user-id", event.playerID).
						Str("receiver-user-id", conn.userID).
						Str("content", string(event.data)).
						Msg("sending data to user")

					conn.send <- event.data
				}
			}
		case <-ctx.Done():
			for id, conn := range s.connections {
				s.logger.Info().Str("user-id", id).Msg("closing connection because root context was canceled")
				conn.socket.Close()
			}
			return
		}
	}
}

func (s *Session) hookBroadcast(event *broadcastMessage) {
	parsed, err := parseEvent(event.data)

	if err != nil {
		if !errors.Is(err, errUnhandledEvent) {
			s.logger.Err(err).Send()
			return
		}

		return
	}

	switch v := parsed.(type) {
	case *loadVideoPayload:
		s.vLock.Lock()
		s.videoID = v.VideoID
		s.vLock.Unlock()
	}
}

func (s *Session) CloseChannels() {
	s.o.Do(func() {
		close(s.broadcast)
		close(s.Register)
		close(s.Unregister)
		close(s.AttachWS)
		close(s.DetachWS)
	})
}
