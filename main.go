package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Config struct {
	VaultPath string
	Port      int
	OllamaURL string
	Model     string
	IndexPath string
}

func main() {
	cfg := Config{}
	flag.StringVar(&cfg.VaultPath, "vault", "./sample_vault", "Path to Obsidian vault markdown directory")
	flag.IntVar(&cfg.Port, "port", 8083, "HTTP service port")
	flag.StringVar(&cfg.OllamaURL, "ollama", "http://localhost:11434", "Ollama API base URL")
	flag.StringVar(&cfg.Model, "model", "nomic-embed-text", "Ollama embedding model")
	flag.StringVar(&cfg.IndexPath, "index", "./vault_index.json", "Persistent vector index file path")
	flag.Parse()

	// Ensure vault directory exists
	if err := os.MkdirAll(cfg.VaultPath, 0755); err != nil {
		log.Fatalf("[Foundry-Graph] Failed to ensure vault directory %s: %v", cfg.VaultPath, err)
	}

	absVault, _ := filepath.Abs(cfg.VaultPath)
	log.Printf("[Foundry-Graph] Initializing Obsidian Vector Engine for vault: %s", absVault)

	store := NewVectorStore(cfg.IndexPath)
	indexer := NewIndexer(absVault, cfg.OllamaURL, cfg.Model, store)

	// Initial index
	if err := indexer.IndexVault(); err != nil {
		log.Printf("[Foundry-Graph] Initial index warning: %v", err)
	}

	// Start file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("[Foundry-Graph] Failed to initialize fsnotify watcher: %v", err)
	}
	defer watcher.Close()

	go startFileWatcher(watcher, absVault, indexer, store)

	// HTTP Router
	mux := http.NewServeMux()

	// CORS wrapper
	withCORS := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next(w, r)
		}
	}

	// 1. Healthcheck
	mux.HandleFunc("/health", withCORS(func(w http.ResponseWriter, r *http.Request) {
		files, chunks := store.Stats()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":        "ok",
			"service":       "foundry-graph",
			"vault_path":    absVault,
			"indexed_files": files,
			"total_chunks":  chunks,
			"timestamp":     time.Now().UTC().Format(time.RFC3339),
		})
	}))

	// 2. D3 Graph Data
	mux.HandleFunc("/api/graph", withCORS(func(w http.ResponseWriter, r *http.Request) {
		graph := store.BuildGraph()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(graph)
	}))

	// 3. Vector Similarity Search
	mux.HandleFunc("/api/search", withCORS(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Query string `json:"query"`
			TopK  int    `json:"top_k"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Query == "" {
			http.Error(w, "Invalid query payload", http.StatusBadRequest)
			return
		}
		if req.TopK <= 0 {
			req.TopK = 5
		}

		queryEmb, err := indexer.GetEmbedding(req.Query)
		if err != nil {
			queryEmb = fallbackEmbedding(req.Query)
		}

		results := store.Search(queryEmb, req.TopK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"query":   req.Query,
			"count":   len(results),
			"results": results,
		})
	}))

	// 4. Force Reindex
	mux.HandleFunc("/api/reindex", withCORS(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		go func() {
			if err := indexer.IndexVault(); err != nil {
				log.Printf("[Foundry-Graph] Manual reindex error: %v", err)
			}
		}()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "triggered",
			"message": "Re-indexing vault in background",
		})
	}))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Server runner
	go func() {
		log.Printf("[Foundry-Graph] Server listening on http://localhost:%d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Foundry-Graph] Server crashed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[Foundry-Graph] Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = store.SaveSnapshot()
	_ = server.Shutdown(ctx)
	log.Println("[Foundry-Graph] Exited successfully.")
}

func startFileWatcher(watcher *fsnotify.Watcher, vaultPath string, idx *Indexer, store *VectorStore) {
	// Add all subdirectories recursively
	_ = filepath.Walk(vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && path != vaultPath {
				return filepath.SkipDir
			}
			_ = watcher.Add(path)
		}
		return nil
	})

	log.Printf("[Watcher] Watching %s for Markdown modifications...", vaultPath)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if strings.HasSuffix(strings.ToLower(event.Name), ".md") {
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					log.Printf("[Watcher] File modified: %s - reindexing", event.Name)
					_ = idx.IndexFile(event.Name)
					_ = store.SaveSnapshot()
				} else if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
					rel, _ := filepath.Rel(vaultPath, event.Name)
					log.Printf("[Watcher] File removed: %s - purging index", rel)
					store.RemoveFile(rel)
					_ = store.SaveSnapshot()
				}
			} else if event.Has(fsnotify.Create) {
				// If a new directory was created, watch it too
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() && !strings.HasPrefix(info.Name(), ".") {
					_ = watcher.Add(event.Name)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[Watcher] Error: %v", err)
		}
	}
}
