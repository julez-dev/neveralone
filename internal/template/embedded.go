package template

import (
	"embed"
)

//go:embed html/*
var HTMLTemplates embed.FS
