package main

import (
	"embed"
	"io/fs"
)

//go:embed frontend/dist
var frontendFS embed.FS

func frontendDistFS() fs.FS {
	sub, _ := fs.Sub(frontendFS, "frontend/dist")
	return sub
}
