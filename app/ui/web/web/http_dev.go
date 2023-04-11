//go:build dev

package web

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
	root := filepath.Join(dir, "app/ui/web/web/assests")

	fs := http.FileServer(http.Dir(root))

	//stripped := http.StripPrefix(root, fs)

	router := http.NewServeMux()

	router.Handle("/", fs)
	return httptest.NewServer(router).URL
}
