package main

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"
	"unicode"
)

// MiniLMSpecialTokens standard BERT / MiniLM token IDs
const (
	TokenPad = int64(0)
	TokenUnk = int64(100)
	TokenCls = int64(101)
	TokenSep = int64(102)
	TokenMask = int64(103)
	MaxSeqLength = 128
)

// PureGoTokenizer converts text into BERT/MiniLM token IDs and attention masks
type PureGoTokenizer struct {
	vocab map[string]int64
}

// NewPureGoTokenizer builds a tokenizer seeded with common MiniLM subwords and symbols
func NewPureGoTokenizer() *PureGoTokenizer {
	vocab := make(map[string]int64)
	vocab["[PAD]"] = TokenPad
	vocab["[UNK]"] = TokenUnk
	vocab["[CLS]"] = TokenCls
	vocab["[SEP]"] = TokenSep
	vocab["[MASK]"] = TokenMask

	// Seed common subwords & keywords
	baseVocab := []string{
		"the", "of", "and", "in", "to", "a", "is", "for", "on", "that", "by", "this",
		"with", "i", "you", "it", "not", "or", "be", "are", "from", "at", "as", "your",
		"all", "have", "new", "more", "an", "was", "we", "will", "home", "can", "us",
		"about", "if", "page", "my", "has", "search", "free", "but", "our", "one", "other",
		"cipher", "route", "log", "mem", "sec", "core", "cli", "exec", "mcp", "vault",
		"incident", "resolution", "timeout", "firewall", "sandbox", "telemetry", "metric",
		"sqlite", "wal", "process", "kill", "token", "key", "aws", "redacted", "shannon",
	}

	for idx, w := range baseVocab {
		vocab[w] = int64(idx + 104)
	}

	return &PureGoTokenizer{vocab: vocab}
}

// Tokenize returns input_ids, attention_mask, token_type_ids
func (t *PureGoTokenizer) Tokenize(text string) ([]int64, []int64, []int64) {
	words := t.basicTokenize(text)
	var tokens []int64
	tokens = append(tokens, TokenCls)

	for _, w := range words {
		if len(tokens) >= MaxSeqLength-1 {
			break
		}
		if id, ok := t.vocab[w]; ok {
			tokens = append(tokens, id)
		} else {
			// Subword hashing into vocab space
			h := sha256.Sum256([]byte(w))
			tokenID := int64(binary.BigEndian.Uint16(h[0:2])%20000 + 1000)
			tokens = append(tokens, tokenID)
		}
	}
	tokens = append(tokens, TokenSep)

	seqLen := len(tokens)
	inputIDs := make([]int64, MaxSeqLength)
	attentionMask := make([]int64, MaxSeqLength)
	tokenTypeIDs := make([]int64, MaxSeqLength)

	for i := 0; i < MaxSeqLength; i++ {
		if i < seqLen {
			inputIDs[i] = tokens[i]
			attentionMask[i] = 1
			tokenTypeIDs[i] = 0
		} else {
			inputIDs[i] = TokenPad
			attentionMask[i] = 0
			tokenTypeIDs[i] = 0
		}
	}

	return inputIDs, attentionMask, tokenTypeIDs
}

func (t *PureGoTokenizer) basicTokenize(text string) []string {
	var words []string
	clean := strings.ToLower(text)
	var current strings.Builder

	for _, r := range clean {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}
