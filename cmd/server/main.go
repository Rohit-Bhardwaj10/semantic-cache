package main

import (
	"fmt"
	"net/http"
)

func main() {
	//a minimal web server
	mux := http.NewServeMux()
	mux.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, World!")
	})
	mux.Handle("/static", http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))
	fmt.Println("Starting server on :8080")
	http.ListenAndServe(":8080", mux)
}

// go build && ./server.exe
