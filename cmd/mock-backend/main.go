package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Request struct {
	Prompt string `json:"prompt"`
}

type Response struct {
	Answer  string `json:"answer"`
	Model   string `json:"model"`
	Latency int64  `json:"latency_ms"`
}

func mockHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Simulate some processing time
	start := time.Now()
	time.Sleep(100 * time.Millisecond)

	resp := Response{
		Answer:  fmt.Sprintf("This is a mock answer for: %s", req.Prompt),
		Model:   "mock-gpt-4o",
		Latency: time.Since(start).Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	http.HandleFunc("/", mockHandler)
	log.Println("Mock Backend starting on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
