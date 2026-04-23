package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type AuctionRecord struct {
	Index     int            `json:"index"`
	Type      string         `json:"type"` // CREATE_AUCTION, PLACE_BID, CLOSE_AUCTION
	Payload   map[string]any `json:"payload"`
	PrevHash  string         `json:"prev_hash"`
	Timestamp string         `json:"timestamp"`
	Hash      string         `json:"hash"`
}

func (r *AuctionRecord) CalcHash() string {
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

func (r *AuctionRecord) Finalize() {
	r.Hash = r.CalcHash()
}
