package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
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

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	producerHost := os.Getenv("PRODUCER_HOST")
	if producerHost == "" {
		producerHost = "producer:8080"
	}
	
	// Get connection check interval from environment or use default
	checkIntervalStr := os.Getenv("CONNECTION_CHECK_INTERVAL")
	checkInterval := 30 // Default 30 seconds
	if checkIntervalStr != "" {
		if val, err := strconv.Atoi(checkIntervalStr); err == nil && val > 0 {
			checkInterval = val
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	
	// Start connection verification in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		verifyConnectionPeriodically(ctx, producerHost, time.Duration(checkInterval)*time.Second)
	}()

	// API Routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := StatusResponse{
			Status:    "Consumer service running",
			Hostname:  hostname,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(status)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	http.HandleFunc("/fetch-messages", func(w http.ResponseWriter, r *http.Request) {
		messages, err := fetchMessages(producerHost)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching messages: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"consumer_hostname": hostname,
			"timestamp":         time.Now().Format(time.RFC3339),
			"messages":          messages,
		})
	})

	http.HandleFunc("/create-message", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var message Message
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		createdMessage, err := createMessage(producerHost, message)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error creating message: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(createdMessage)
	})

	log.Printf("Consumer service starting on port %s", port)
	log.Printf("Connected to producer service at %s", producerHost)
	log.Printf("Connection check interval: %d seconds", checkInterval)
	
	// This will block until the server encounters an error
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Printf("HTTP server error: %v", err)
		cancel() // Cancel the connection verification goroutine
	}
	
	// Wait for background goroutines to complete
	wg.Wait()
}

func fetchMessages(producerHost string) ([]Message, error) {
	url := fmt.Sprintf("http://%s/messages", producerHost)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to producer service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("producer service returned non-200 status code: %d - %s", resp.StatusCode, string(body))
	}

	var messages []Message
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return messages, nil
}

func createMessage(producerHost string, message Message) (*Message, error) {
	url := fmt.Sprintf("http://%s/messages", producerHost)
	
	jsonData, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %v", err)
	}
	
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to producer service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("producer service returned non-201 status code: %d - %s", resp.StatusCode, string(body))
	}

	var createdMessage Message
	if err := json.NewDecoder(resp.Body).Decode(&createdMessage); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &createdMessage, nil
}

// verifyConnectionPeriodically checks the connection to the producer service at regular intervals
func verifyConnectionPeriodically(ctx context.Context, producerHost string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	// Run an initial check immediately
	checkProducerConnection(producerHost)
	
	for {
		select {
		case <-ticker.C:
			checkProducerConnection(producerHost)
		case <-ctx.Done():
			log.Println("Connection verification stopped")
			return
		}
	}
}

// checkProducerConnection verifies the connection to the producer service
func checkProducerConnection(producerHost string) {
	start := time.Now()
	url := fmt.Sprintf("http://%s/health", producerHost)
	
	// Set a timeout for the request
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("[CONNECTION CHECK] Failed to connect to producer at %s: %v", producerHost, err)
		return
	}
	defer resp.Body.Close()
	
	elapsed := time.Since(start).Milliseconds()
	
	if resp.StatusCode == http.StatusOK {
		log.Printf("[CONNECTION CHECK] Successfully connected to producer at %s (status: %d, latency: %dms)", 
			producerHost, resp.StatusCode, elapsed)
	} else {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[CONNECTION CHECK] Producer service returned unexpected status: %d, body: %s, latency: %dms", 
			resp.StatusCode, string(body), elapsed)
	}
}