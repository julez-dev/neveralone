package static

import (
	"embed"
)

//go:embed static/*
var StaticFiles embed.FS
