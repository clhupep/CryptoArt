package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

type AuctionChain struct {
	Records []AuctionRecord `json:"-"`
	mu      sync.RWMutex
	File    string
}

// Вспомогательная структура для JSON (исключает mutex)
type chainStorage struct {
	Records []AuctionRecord `json:"chain"`
}

func NewChain(file string) (*AuctionChain, error) {
	chain := &AuctionChain{File: file}
	if _, err := os.Stat(file); err == nil {
		if err := chain.load(); err != nil {
			return nil, err
		}
		if !chain.validateInternal() {
			return nil, errors.New("❌ Целостность цепи нарушена при загрузке")
		}
	} else {
		chain.mu.Lock()
		chain.initGenesis()
		chain.saveInternal()
		chain.mu.Unlock()
	}
	return chain, nil
}

func (c *AuctionChain) initGenesis() {
	genesis := AuctionRecord{
		Index: 0, Type: "GENESIS",
		Payload:   map[string]any{"message": "Art Auction Chain Initialized"},
		PrevHash:  "0",
		Timestamp: time.Now().Format(time.RFC3339),
	}
	genesis.Finalize()
	c.Records = append(c.Records, genesis)
}

func (c *AuctionChain) AddRecord(recordType string, payload map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prevHash := "0"
	if len(c.Records) > 0 {
		prevHash = c.Records[len(c.Records)-1].Hash
	}

	rec := AuctionRecord{
		Index:     len(c.Records),
		Type:      recordType,
		Payload:   payload,
		PrevHash:  prevHash,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	rec.Finalize()
	c.Records = append(c.Records, rec)
	c.saveInternal()
	fmt.Println("✅ Запись добавлена и сохранена.")
}

func (c *AuctionChain) Validate() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.validateInternal()
}

func (c *AuctionChain) validateInternal() bool {
	for i, rec := range c.Records {
		if rec.Hash != rec.CalcHash() {
			return false
		}
		if i == 0 {
			if rec.PrevHash != "0" {
				return false
			}
		} else {
			if rec.PrevHash != c.Records[i-1].Hash {
				return false
			}
		}
	}
	return true
}

func (c *AuctionChain) saveInternal() {
	data, _ := json.MarshalIndent(chainStorage{Records: c.Records}, "", "  ")
	os.WriteFile(c.File, data, 0644)
}

func (c *AuctionChain) load() error {
	data, err := os.ReadFile(c.File)
	if err != nil {
		return err
	}
	var storage chainStorage
	if err := json.Unmarshal(data, &storage); err != nil {
		return err
	}
	c.Records = storage.Records
	return nil
}

func (c *AuctionChain) GetActiveAuctions() []map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	opened := make(map[string]map[string]any)
	for _, rec := range c.Records {
		switch rec.Type {
		case "CREATE_AUCTION":
			pid := rec.Payload["painting_id"].(string)
			opened[pid] = rec.Payload
			opened[pid]["status"] = "open"
		case "CLOSE_AUCTION":
			pid := rec.Payload["painting_id"].(string)
			if a, ok := opened[pid]; ok {
				a["status"] = "closed"
				a["winner"] = rec.Payload["winner"]
				a["final_price"] = rec.Payload["final_price"]
			}
		}
	}
	res := make([]map[string]any, 0, len(opened))
	for _, a := range opened {
		res = append(res, a)
	}
	return res
}

func (c *AuctionChain) SimulateHack(index int, newPayload map[string]any) (AuctionRecord, string, bool) {
	c.mu.RLock()
	if index < 1 || index >= len(c.Records) {
		c.mu.RUnlock()
		return AuctionRecord{}, "invalid_index", false
	}
	rec := c.Records[index]
	c.mu.RUnlock()

	rec.Payload = newPayload
	rec.Hash = rec.CalcHash()

	reason := "none"
	nextBroken := false
	if rec.PrevHash != c.Records[index-1].Hash {
		reason = "prev_hash_link_broken"
	}
	if index+1 < len(c.Records) && c.Records[index+1].PrevHash != rec.Hash {
		reason = "next_block_reference_broken"
		nextBroken = true
	}
	return rec, reason, nextBroken
}

// GetCurrentPrice возвращает текущую максимальную ставку и флаг "аукцион открыт"
func (c *AuctionChain) GetCurrentPrice(paintingID string) (float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var currentPrice float64
	var isOpen bool

	for _, rec := range c.Records {
		if rec.Payload["painting_id"] != paintingID {
			continue
		}
		switch rec.Type {
		case "CREATE_AUCTION":
			isOpen = true
			if v, ok := rec.Payload["start_price"].(float64); ok {
				currentPrice = v
			}
		case "CLOSE_AUCTION":
			isOpen = false
		case "PLACE_BID":
			if isOpen {
				if v, ok := rec.Payload["amount"].(float64); ok && v > currentPrice {
					currentPrice = v
				}
			}
		}
	}
	return currentPrice, isOpen
}
