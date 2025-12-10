package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Node struct {
	ID    string            `json:"id"`
	Props map[string]string `json:"props,omitempty"`
}

type Edge struct {
	From  string `json:"From"`
	To    string `json:"To"`
	Label string `json:"Label,omitempty"`
}

type GraphStore struct {
	nodes map[string]Node
	edges []Edge
	mu    sync.RWMutex
}

func NewGraphStore() *GraphStore {
	return &GraphStore{
		nodes: make(map[string]Node),
		edges: make([]Edge, 0),
	}
}

func (gs *GraphStore) AddNode(n Node) error {
	if n.ID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.nodes[n.ID] = n
	return nil
}

func (gs *GraphStore) AddEdge(e Edge) error {
	if e.From == "" || e.To == "" {
		return fmt.Errorf("edge From and To cannot be empty")
	}
	gs.mu.Lock()
	defer gs.mu.Unlock()
	
	// Validate nodes exist
	if _, exists := gs.nodes[e.From]; !exists {
		return fmt.Errorf("source node %s does not exist", e.From)
	}
	if _, exists := gs.nodes[e.To]; !exists {
		return fmt.Errorf("target node %s does not exist", e.To)
	}
	
	gs.edges = append(gs.edges, e)
	return nil
}

func (gs *GraphStore) GetNode(id string) (Node, bool) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	n, exists := gs.nodes[id]
	return n, exists
}

func (gs *GraphStore) Export() map[string]interface{} {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return map[string]interface{}{
		"nodes": gs.nodes,
		"edges": gs.edges,
		"stats": map[string]int{
			"node_count": len(gs.nodes),
			"edge_count": len(gs.edges),
		},
	}
}

func (gs *GraphStore) Clear() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.nodes = make(map[string]Node)
	gs.edges = make([]Edge, 0)
}

// HTTP Handlers
func (gs *GraphStore) handleAddNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var n Node
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if err := gs.AddNode(n); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "id": n.ID})
}

func (gs *GraphStore) handleAddEdge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var e Edge
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if err := gs.AddEdge(e); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (gs *GraphStore) handleGetNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Node ID required", http.StatusBadRequest)
		return
	}

	node, exists := gs.GetNode(id)
	if !exists {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(node)
}

func (gs *GraphStore) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gs.Export())
}

func (gs *GraphStore) handleClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	gs.Clear()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cleared"})
}

func (gs *GraphStore) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
		log.Printf("Completed in %v", time.Since(start))
	})
}

func main() {
	gs := NewGraphStore()

	mux := http.NewServeMux()
	mux.HandleFunc("/node", gs.handleAddNode)
	mux.HandleFunc("/node/get", gs.handleGetNode)
	mux.HandleFunc("/edge", gs.handleAddEdge)
	mux.HandleFunc("/export", gs.handleExport)
	mux.HandleFunc("/clear", gs.handleClear)
	mux.HandleFunc("/health", gs.handleHealth)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      corsMiddleware(loggingMiddleware(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Server is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Server forced to shutdown: %v", err)
		}
		close(done)
	}()

	log.Printf("Graph store server starting on port %s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}

	<-done
	log.Println("Server stopped")
}