package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/test", func(writer http.ResponseWriter, request *http.Request) {

		_, _ = writer.Write([]byte(fmt.Sprintf("A sample from host at %s - V2", time.Now())))
	})
	_ = http.ListenAndServe(":8082", nil)
}
