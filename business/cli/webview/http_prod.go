//go:build !dev

package webview

import (
	"embed"
	"io/fs"
	"net/http"
	"net/http/httptest"
)

//go:embed webroot
var eroot embed.FS

func RunWebserver() string {
	fSys, err := fs.Sub(eroot, "webroot")
	if err != nil {
		panic(err)
	}

	staticServer := http.FileServer(http.FS(fSys))
	//stripped := http.StripPrefix("webview", staticServer)
	router := http.NewServeMux()
	router.Handle("/", staticServer)
	return httptest.NewServer(router).URL
}
