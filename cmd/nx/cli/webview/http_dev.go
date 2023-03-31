//go:build dev

package webview

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
)

func RunWebserver() string {

	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root := filepath.Join(dir, "cmd/nx/cli/webview/webroot")

	fs := http.FileServer(http.Dir(root))

	//stripped := http.StripPrefix(root, fs)

	router := http.NewServeMux()

	router.Handle("/", fs)
	return httptest.NewServer(router).URL
}
