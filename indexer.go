package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	wikiLinkRegex = regexp.MustCompile(`\[\[([^\]\|]+)(?:\|[^\]]+)?\]\]`)
	mdLinkRegex   = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	tagRegex      = regexp.MustCompile(`#([a-zA-Z0-9_\-]+)`)
	headingRegex  = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
)

type Indexer struct {
	vaultPath  string
	vectorizer Vectorizer
	store      *VectorStore
}

func NewIndexer(vaultPath string, store *VectorStore, vec Vectorizer) *Indexer {
	if vec == nil {
		vec = NewEmbeddedMiniLMVectorizer()
	}
	return &Indexer{
		vaultPath:  vaultPath,
		vectorizer: vec,
		store:      store,
	}
}

// IndexVault scans all markdown files in vaultPath recursively
func (idx *Indexer) IndexVault() error {
	log.Printf("[Indexer] Starting full indexing of vault: %s", idx.vaultPath)
	startTime := time.Now()
	var count int

	err := filepath.Walk(idx.vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip hidden dirs (.git, .obsidian, etc.)
			if strings.HasPrefix(info.Name(), ".") && path != idx.vaultPath {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			if err := idx.IndexFile(path); err != nil {
				log.Printf("[Indexer] Error indexing file %s: %v", path, err)
			} else {
				count++
			}
		}
		return nil
	})

	files, chunks := idx.store.Stats()
	log.Printf("[Indexer] Finished indexing %d files (%d total in store, %d chunks) in %v", count, files, chunks, time.Since(startTime))
	_ = idx.store.SaveSnapshot()
	return err
}

// IndexFile parses, chunks, embeds, and stores a single file
func (idx *Indexer) IndexFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	relPath, err := filepath.Rel(idx.vaultPath, path)
	if err != nil {
		relPath = path
	}

	// 1. Extract Links & Tags
	outLinks := extractLinks(content)
	tags := extractTags(content)

	// 2. Chunk markdown by headings and paragraphs
	rawChunks := chunkMarkdown(relPath, content)

	// 3. Generate embeddings
	var chunks []Chunk
	for i, rc := range rawChunks {
		chunkID := fmt.Sprintf("%s#%d", relPath, i)
		emb, err := idx.GetEmbedding(rc.Content)
		if err != nil {
			log.Printf("[Indexer] Embedding warning for %s: %v; falling back to deterministic vector", chunkID, err)
			emb = fallbackEmbedding(rc.Content)
		}

		chunks = append(chunks, Chunk{
			ID:        chunkID,
			File:      relPath,
			Title:     rc.Title,
			Heading:   rc.Heading,
			Content:   rc.Content,
			Embedding: emb,
			Tags:      tags,
			UpdatedAt: time.Now(),
		})
	}

	idx.store.UpsertFile(relPath, chunks, outLinks)
	return nil
}

type RawChunk struct {
	Title   string
	Heading string
	Content string
}

func chunkMarkdown(filePath, content string) []RawChunk {
	lines := strings.Split(content, "\n")
	var chunks []RawChunk

	docTitle := cleanFileName(filePath)
	currentHeading := "Overview"
	var currentParagraph []string

	flush := func() {
		text := strings.TrimSpace(strings.Join(currentParagraph, "\n"))
		if len(text) > 30 {
			chunks = append(chunks, RawChunk{
				Title:   docTitle,
				Heading: currentHeading,
				Content: text,
			})
		}
		currentParagraph = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if match := headingRegex.FindStringSubmatch(trimmed); len(match) == 3 {
			flush()
			currentHeading = match[2]
			if docTitle == cleanFileName(filePath) && match[1] == "#" {
				docTitle = match[2]
			}
			continue
		}

		if trimmed == "" {
			if len(strings.Join(currentParagraph, "\n")) > 400 {
				flush()
			}
		} else {
			currentParagraph = append(currentParagraph, line)
		}
	}
	flush()

	if len(chunks) == 0 && len(strings.TrimSpace(content)) > 0 {
		chunks = append(chunks, RawChunk{
			Title:   docTitle,
			Heading: "Full Document",
			Content: strings.TrimSpace(content),
		})
	}

	return chunks
}

func extractLinks(content string) []string {
	linkSet := make(map[string]bool)

	// [[wikilinks]]
	for _, match := range wikiLinkRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			target := strings.TrimSpace(match[1])
			if target != "" {
				linkSet[target] = true
			}
		}
	}

	// [text](path.md)
	for _, match := range mdLinkRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 2 {
			url := strings.TrimSpace(match[2])
			if strings.HasSuffix(url, ".md") {
				clean := cleanFileName(url)
				linkSet[clean] = true
			}
		}
	}

	var links []string
	for l := range linkSet {
		links = append(links, l)
	}
	return links
}

func extractTags(content string) []string {
	tagSet := make(map[string]bool)
	for _, match := range tagRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			tagSet[match[1]] = true
		}
	}
	var tags []string
	for t := range tagSet {
		tags = append(tags, t)
	}
	return tags
}

// GetEmbedding executes sub-millisecond vectorization via the embedded ONNX/pure-Go vectorizer
func (idx *Indexer) GetEmbedding(text string) ([]float64, error) {
	if idx.vectorizer != nil {
		return idx.vectorizer.Embed(text)
	}
	return fallbackEmbedding(text), nil
}

// fallbackEmbedding generates a normalized 128-dimensional embedding from text n-grams
func fallbackEmbedding(text string) []float64 {
	const dims = 128
	vec := make([]float64, dims)
	words := strings.Fields(strings.ToLower(text))

	for _, w := range words {
		h := sha256.Sum256([]byte(w))
		for i := 0; i < dims; i++ {
			byteIdx := (i * 2) % 32
			val := float64(binary.BigEndian.Uint16(h[byteIdx:byteIdx+2])) / 65535.0
			vec[i] += (val - 0.5)
		}
	}

	// L2 normalization
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range vec {
			vec[i] /= norm
		}
	}
	return vec
}
