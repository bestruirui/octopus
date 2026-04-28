package semantic_cache

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
)

// GenerateEmbedding creates a 64-dimension embedding from the input text.
// Currently uses a SHA-256 hash for deterministic caching — identical
// requests produce identical embeddings, while similar (but not identical)
// requests will not match. A future upgrade can replace this with a real
// embedding model via the system's embedding relay pipeline.
func GenerateEmbedding(text string) []float64 {
	h := sha256.Sum256([]byte(text))
	embedding := make([]float64, 64)
	for i := 0; i < 32; i++ {
		// Each byte pair → one float64 in [-1, 1]
		val := float64(binary.BigEndian.Uint16(h[i*2 : i*2+2]))
		embedding[i] = val/32768.0 - 1.0
	}
	// Normalize to unit vector
	var norm float64
	for _, v := range embedding {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range embedding {
			embedding[i] /= norm
		}
	}
	return embedding
}
