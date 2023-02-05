package handler

import (
	"fmt"
	"github.com/julez-dev/neveralone/internal/party"
	"html/template"
	"io"
	"io/fs"
)

type TemplateExecuter interface {
	ExecuteTemplate(io.Writer, string, any) error
}

type FSExecuter struct {
	tmpl *template.Template
}

func NewFSExecuter(fs fs.FS, glob string) (*FSExecuter, error) {
	tmpl, err := template.ParseFS(fs, glob)

	if err != nil {
		return nil, err
	}

	return &FSExecuter{tmpl: tmpl}, nil
}

func (t *FSExecuter) ExecuteTemplate(w io.Writer, name string, data any) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

type DebuggerExecuter struct {
	glob string
}

func NewDebuggerExecuter(glob string) *DebuggerExecuter {
	return &DebuggerExecuter{glob: glob}
}

func (d *DebuggerExecuter) ExecuteTemplate(w io.Writer, name string, data any) error {
	t, err := template.ParseGlob(d.glob)

	if err != nil {
		return err
	}

	return t.ExecuteTemplate(w, name, data)
}

type Template struct {
	executer TemplateExecuter
	store    sessionStore
}

type sessionStore interface {
	Get(string) (*party.Session, bool)
	GetAll() map[string]*party.Session
}

func NewTemplate(executer TemplateExecuter, store sessionStore) (*Template, error) {
	return &Template{
		executer: executer,
		store:    store,
	}, nil
}

func (t *Template) ServeHome(writer io.Writer, loggedUser *party.User) error {
	type sessionData struct {
		ID     string
		Player []*party.Player
	}

	var (
		userSessions   []*sessionData
		publicSessions []*sessionData
	)

	session := t.store.GetAll()

OUTER:
	for _, session := range session {
		users := session.GetPlayersCopy()

		for _, user := range users {
			// If user is member of this room
			if user.User.ID == loggedUser.ID {
				userSessions = append(userSessions, &sessionData{
					ID:     session.ID.String(),
					Player: users,
				})
				continue OUTER
			}
		}

		cfg := session.GetConfig()
		if cfg.Visibility == party.PublicLobby {
			publicSessions = append(publicSessions, &sessionData{
				ID:     session.ID.String(),
				Player: users,
			})
		}
	}

	data := struct {
		User           *party.User
		Sessions       []*sessionData
		PublicSessions []*sessionData
	}{
		User:           loggedUser,
		Sessions:       userSessions,
		PublicSessions: publicSessions,
	}

	return t.executer.ExecuteTemplate(writer, "index.gohtml", data)
}

func (t *Template) ServeParty(writer io.Writer, id string, user *party.User) error {
	session, ok := t.store.Get(id)

	if !ok {
		return fmt.Errorf("could not find session")
	}

	state := session.GetCurrentState()
	cfg := session.GetConfig()
	player := session.GetPlayersCopy()

	showVideoInput := true

	if cfg.AllowOnlyHost {
		var hostID string
		for _, player := range player {
			if player.IsHost {
				hostID = player.User.ID.String()
				break
			}
		}

		if hostID != user.ID.String() {
			showVideoInput = false
		}
	}

	data := struct {
		User           *party.User
		Player         []*party.Player
		State          *party.VideoStateSnapshot
		ShowVideoInput bool
	}{
		User:           user,
		Player:         player,
		State:          state,
		ShowVideoInput: showVideoInput,
	}

	return t.executer.ExecuteTemplate(writer, "party.gohtml", data)
}

func (t *Template) ServeCreateParty(writer io.Writer, user *party.User) error {
	data := struct {
		User *party.User
	}{
		User: user,
	}

	return t.executer.ExecuteTemplate(writer, "create_party.gohtml", data)
}

func (t *Template) ServeJoinPartyPassword(writer io.Writer, user *party.User, sessionID string, wrongPassword bool) error {
	data := struct {
		User          *party.User
		SessionID     string
		WrongPassword bool
	}{
		User:          user,
		SessionID:     sessionID,
		WrongPassword: wrongPassword,
	}

	return t.executer.ExecuteTemplate(writer, "login.gohtml", data)
}
