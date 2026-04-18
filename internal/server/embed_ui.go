//go:build embedui

package server

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var embeddedUIDist embed.FS

func embeddedUIAvailable() bool {
	sub, err := fs.Sub(embeddedUIDist, "dist")
	if err != nil {
		return false
	}
	_, err = fs.Stat(sub, "index.html")
	return err == nil
}

func spaHandlerEmbedded(basePath string) http.Handler {
	sub, err := fs.Sub(embeddedUIDist, "dist")
	if err != nil {
		return uiPlaceholderHandler()
	}
	return spaHandlerFromFS(sub, basePath)
}
