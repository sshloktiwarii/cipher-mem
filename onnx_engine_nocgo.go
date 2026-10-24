//go:build !cgo

package main

import (
	"log"
)

// ONNXEmbeddingEngine fallback implementation when compiling without CGO
type ONNXEmbeddingEngine struct {
	modelPath string
	fallback  *EmbeddedMiniLMVectorizer
}

// NewONNXEmbeddingEngine initializes the pure-Go MiniLM engine for static non-CGO distributions
func NewONNXEmbeddingEngine(modelPath string, sharedLibPath string) *ONNXEmbeddingEngine {
	log.Printf("[ONNX] Static zero-dependency distribution (non-CGO): initializing embedded MiniLM-384 offline vectorizer")
	return &ONNXEmbeddingEngine{
		modelPath: modelPath,
		fallback:  NewEmbeddedMiniLMVectorizer(),
	}
}

func (o *ONNXEmbeddingEngine) Dimension() int {
	return MiniLMDimension
}

func (o *ONNXEmbeddingEngine) EngineName() string {
	return "embedded-minilm-384 (offline zero-dependency)"
}

func (o *ONNXEmbeddingEngine) Embed(text string) ([]float64, error) {
	return o.fallback.Embed(text)
}
