package handler

import (
	"fmt"
	"github.com/julez-dev/neveralone/internal/party"
	"html/template"
	"io"
	"io/fs"
)

type templateExecuter interface {
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
	executer templateExecuter
	store    sessionStore
}

type sessionStore interface {
	Get(string) (*party.Session, bool)
	GetAll() map[string]*party.Session
}

func NewTemplate(executer templateExecuter, store sessionStore) (*Template, error) {
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

	var allSessionData []*sessionData

	session := t.store.GetAll()

	for _, session := range session {
		users := session.GetPlayersCopy()

		for _, user := range users {
			// If user is member of this room
			if user.User.ID == loggedUser.ID {
				allSessionData = append(allSessionData, &sessionData{
					ID:     session.ID.String(),
					Player: users,
				})
				break
			}

		}
	}

	data := struct {
		User     *party.User
		Sessions []*sessionData
	}{
		User:     loggedUser,
		Sessions: allSessionData,
	}

	return t.executer.ExecuteTemplate(writer, "index.gohtml", data)
}

func (t *Template) ServeParty(writer io.Writer, id string, user *party.User) error {
	session, ok := t.store.Get(id)

	if !ok {
		return fmt.Errorf("could not find session")
	}

	state := session.GetCurrentState()

	data := struct {
		User   *party.User
		Player []*party.Player
		State  *party.VideoStateSnapshot
	}{
		User:   user,
		Player: session.GetPlayersCopy(),
		State:  state,
	}

	return t.executer.ExecuteTemplate(writer, "party.gohtml", data)
}
