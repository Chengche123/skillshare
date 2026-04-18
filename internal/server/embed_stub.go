//go:build !embedui

package server

import "net/http"

func embeddedUIAvailable() bool {
	return false
}

func spaHandlerEmbedded(basePath string) http.Handler {
	return uiPlaceholderHandler()
}
