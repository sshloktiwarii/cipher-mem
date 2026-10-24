package main

import (
	"encoding/json"
	"math"
	"os"
	"sort"
	"sync"
	"time"
)

// Chunk represents an embedded markdown snippet with metadata
type Chunk struct {
	ID        string    `json:"id"`
	File      string    `json:"file"`
	Title     string    `json:"title"`
	Heading   string    `json:"heading"`
	Content   string    `json:"content"`
	Embedding []float64 `json:"embedding,omitempty"`
	Tags      []string  `json:"tags"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GraphNode represents a node in the force-directed graph
type GraphNode struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Path      string   `json:"path"`
	Group     string   `json:"group"`
	Tags      []string `json:"tags"`
	Size      int      `json:"size"`
	ChunkCount int     `json:"chunk_count"`
}

// GraphLink represents a directed connection between two nodes
type GraphLink struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Value  float64 `json:"value"`
	Type   string  `json:"type"` // "wikilink", "markdown_link", "semantic_affinity"
}

// GraphData is the D3-compatible network payload
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Links []GraphLink `json:"links"`
}

// SearchResult matches query embeddings
type SearchResult struct {
	Chunk      Chunk   `json:"chunk"`
	Similarity float64 `json:"similarity"`
}

// VectorStore holds all indexed chunks and document connections
type VectorStore struct {
	mu        sync.RWMutex
	chunks    map[string]Chunk
	files     map[string][]string // file path -> chunk IDs
	docLinks  map[string][]string // source file -> target file names
	indexPath string
}

func NewVectorStore(indexPath string) *VectorStore {
	return &VectorStore{
		chunks:    make(map[string]Chunk),
		files:     make(map[string][]string),
		docLinks:  make(map[string][]string),
		indexPath: indexPath,
	}
}

// UpsertFile atomically replaces all chunks for a file
func (vs *VectorStore) UpsertFile(filePath string, chunks []Chunk, outLinks []string) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Remove old chunks
	if oldIDs, exists := vs.files[filePath]; exists {
		for _, id := range oldIDs {
			delete(vs.chunks, id)
		}
	}

	newIDs := make([]string, 0, len(chunks))
	for _, c := range chunks {
		vs.chunks[c.ID] = c
		newIDs = append(newIDs, c.ID)
	}

	vs.files[filePath] = newIDs
	vs.docLinks[filePath] = outLinks
}

// RemoveFile deletes a file and its chunks from index
func (vs *VectorStore) RemoveFile(filePath string) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if ids, exists := vs.files[filePath]; exists {
		for _, id := range ids {
			delete(vs.chunks, id)
		}
		delete(vs.files, filePath)
		delete(vs.docLinks, filePath)
	}
}

// CosineSimilarity computes the cosine of the angle between two vectors
func CosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// Search searches across all indexed chunks by cosine similarity
func (vs *VectorStore) Search(queryVec []float64, topK int) []SearchResult {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	results := make([]SearchResult, 0, len(vs.chunks))
	for _, chunk := range vs.chunks {
		sim := CosineSimilarity(queryVec, chunk.Embedding)
		results = append(results, SearchResult{
			Chunk:      chunk,
			Similarity: sim,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if len(results) > topK {
		return results[:topK]
	}
	return results
}

// BuildGraph constructs the nodes and edges for force-directed visualization
func (vs *VectorStore) BuildGraph() GraphData {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	nodeMap := make(map[string]GraphNode)
	var links []GraphLink

	// 1. Create file nodes
	for file, chunkIDs := range vs.files {
		group := "vault"
		if len(chunkIDs) > 0 {
			firstChunk := vs.chunks[chunkIDs[0]]
			if len(firstChunk.Tags) > 0 {
				group = firstChunk.Tags[0]
			}
		}

		nodeMap[file] = GraphNode{
			ID:         file,
			Name:       cleanFileName(file),
			Path:       file,
			Group:      group,
			Size:       len(chunkIDs)*4 + 8,
			ChunkCount: len(chunkIDs),
		}
	}

	// 2. Add explicit wikilinks/markdown edges
	for srcFile, targets := range vs.docLinks {
		for _, target := range targets {
			// Find matching target node
			for file := range vs.files {
				if cleanFileName(file) == target || file == target {
					links = append(links, GraphLink{
						Source: srcFile,
						Target: file,
						Value:  1.0,
						Type:   "wikilink",
					})
					break
				}
			}
		}
	}

	var nodes []GraphNode
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}

	return GraphData{
		Nodes: nodes,
		Links: links,
	}
}

func cleanFileName(p string) string {
	// Strip directory and extension for node display
	name := p
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			name = p[i+1:]
			break
		}
	}
	if len(name) > 3 && name[len(name)-3:] == ".md" {
		name = name[:len(name)-3]
	}
	return name
}

// Stats returns index metrics
func (vs *VectorStore) Stats() (int, int) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return len(vs.files), len(vs.chunks)
}

// SaveSnapshot serializes index to disk
func (vs *VectorStore) SaveSnapshot() error {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	data, err := json.MarshalIndent(vs.chunks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(vs.indexPath, data, 0644)
}
