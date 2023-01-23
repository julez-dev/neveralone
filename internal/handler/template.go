package handler

import (
	"fmt"
	"github.com/julez-dev/neveralone/internal/party"
	"html/template"
	"io"
)
import ctemplate "github.com/julez-dev/neveralone/internal/template"

type Template struct {
	tmpl  *template.Template
	store sessionStore
}

type sessionStore interface {
	Get(string) (*party.Session, bool)
}

func NewTemplate(store sessionStore) (*Template, error) {
	tmpl, err := template.ParseFS(ctemplate.HTMLTemplates, "html/*")

	if err != nil {
		return nil, err
	}

	return &Template{
		tmpl:  tmpl,
		store: store,
	}, nil
}

func (t *Template) ServeTemplate(writer io.Writer, data any) error {
	return t.tmpl.Execute(writer, data)
}

func (t *Template) ServeHome(writer io.Writer) error {
	return t.tmpl.ExecuteTemplate(writer, "index.html", nil)
}

func (t *Template) ServeParty(writer io.Writer, id string) error {
	session, ok := t.store.Get(id)

	if !ok {
		return fmt.Errorf("could not find session")
	}

	_ = session
	data := struct {
		Player []*party.Player
	}{
		Player: session.GetPlayersCopy(),
	}

	return t.tmpl.ExecuteTemplate(writer, "party.html", data)
}
