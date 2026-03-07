package main

import (
	"embed"
	"io/fs"
)

//go:embed web
var frontendFS embed.FS

func frontendDistFS() fs.FS {
	sub, _ := fs.Sub(frontendFS, "web")
	return sub
}
