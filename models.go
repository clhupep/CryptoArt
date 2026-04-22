package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// AuctionRecord аналог ChainRecord из Python
type AuctionRecord struct {
	Index     int            `json:"index"`
	Type      string         `json:"type"` // CREATE_AUCTION, PLACE_BID, CLOSE_AUCTION
	Payload   map[string]any `json:"payload"`
	PrevHash  string         `json:"prev_hash"`
	Timestamp string         `json:"timestamp"`
	Hash      string         `json:"hash"`
}

// CalcHash вычисляет SHA-256 от отсортированного JSON (детерминировано)
func (r *AuctionRecord) CalcHash() string {
	// В Go 1.12+ json.Marshal сортирует ключи мап автоматически
	data, _ := json.Marshal(map[string]any{
		"index":     r.Index,
		"type":      r.Type,
		"payload":   r.Payload,
		"prev_hash": r.PrevHash,
		"timestamp": r.Timestamp,
	})
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// Finalize аналог finalize() из Python
func (r *AuctionRecord) Finalize() {
	r.Hash = r.CalcHash()
}
