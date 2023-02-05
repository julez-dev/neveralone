package static

import (
	"embed"
)

//go:embed static/*
var StaticFiles embed.FS

//go:embed static/favicon.ico
var IconFile []byte
