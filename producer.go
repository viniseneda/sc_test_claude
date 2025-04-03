package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type Message struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type StatusResponse struct {
	Status    string `json:"status"`
	Hostname  string `json:"hostname"`
	Timestamp string `json:"timestamp"`
}

var messages []Message

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// API Routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := StatusResponse{
			Status:    "Producer service running",
			Hostname:  hostname,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(status)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	http.HandleFunc("/messages", handleMessages)
	
	log.Printf("Producer service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		json.NewEncoder(w).Encode(messages)
	case "POST":
		var message Message
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		message.ID = fmt.Sprintf("%d", time.Now().UnixNano())
		message.Timestamp = time.Now()
		messages = append(messages, message)
		
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(message)
		
		log.Printf("New message created: %s", message.ID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
