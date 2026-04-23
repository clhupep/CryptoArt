package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

var chain *AuctionChain

func main() {
	var err error
	chain, err = NewChain("chain_storage.json")
	if err != nil {
		log.Fatal("Ошибка инициализации цепи: ", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/chain", handleGetChain)
	mux.HandleFunc("POST /api/auction/create", handleCreateAuction)
	mux.HandleFunc("POST /api/auction/bid", handlePlaceBid)
	mux.HandleFunc("POST /api/auction/close", handleCloseAuction)
	mux.HandleFunc("GET /api/validate", handleValidate)

	mux.Handle("/", http.FileServer(http.Dir("static")))

	fmt.Println("Art Auction Blockchain running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func respondJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func ensureChainIntegrity(w http.ResponseWriter) bool {
	valid, err := chain.ReloadAndValidate()
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": err.Error(), "status": "error"})
		return false
	}
	if !valid {
		respondJSON(w, 503, map[string]string{
			"error":  "Цепь повреждена! Обнаружена несанкционированная модификация.",
			"status": "compromised",
		})
		return false
	}
	return true
}

func handleCreateAuction(w http.ResponseWriter, r *http.Request) {
	if !ensureChainIntegrity(w) {
		return
	}

	var req struct {
		PaintingID string  `json:"painting_id"`
		Title      string  `json:"title"`
		Owner      string  `json:"owner"`
		StartPrice float64 `json:"start_price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}

	chain.AddRecord("CREATE_AUCTION", map[string]any{
		"painting_id": req.PaintingID, "title": req.Title, "owner": req.Owner,
		"start_price": req.StartPrice, "current_bid": req.StartPrice, "bidder": req.Owner,
	})
	respondJSON(w, 200, map[string]string{"status": "ok"})
}

func handlePlaceBid(w http.ResponseWriter, r *http.Request) {
	if !ensureChainIntegrity(w) {
		return
	}

	var req struct {
		PaintingID string  `json:"painting_id"`
		Bidder     string  `json:"bidder"`
		Amount     float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}

	if req.Amount <= 0 {
		respondJSON(w, 400, map[string]string{"error": "Сумма ставки должна быть больше нуля"})
		return
	}

	currentPrice, isOpen := chain.GetCurrentPrice(req.PaintingID)
	if !isOpen {
		respondJSON(w, 400, map[string]string{"error": "Аукцион не найден или уже закрыт"})
		return
	}
	if req.Amount <= currentPrice {
		respondJSON(w, 400, map[string]string{"error": fmt.Sprintf("❌ Ставка %.2f ≤ текущей цене (%.2f). Повысьте ставку!", req.Amount, currentPrice)})
		return
	}

	chain.AddRecord("PLACE_BID", map[string]any{
		"painting_id": req.PaintingID, "bidder": req.Bidder, "amount": req.Amount,
	})
	respondJSON(w, 200, map[string]string{"status": "ok"})
}

func handleCloseAuction(w http.ResponseWriter, r *http.Request) {
	if !ensureChainIntegrity(w) {
		return
	}

	var req struct {
		PaintingID string `json:"painting_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}

	var lastBidder string = "System"
	var lastAmount float64 = 0
	chain.mu.RLock()
	for i := len(chain.Records) - 1; i >= 0; i-- {
		rec := chain.Records[i]
		if rec.Type == "PLACE_BID" && rec.Payload["painting_id"] == req.PaintingID {
			lastBidder = rec.Payload["bidder"].(string)
			lastAmount = rec.Payload["amount"].(float64)
			break
		}
	}
	chain.mu.RUnlock()

	chain.AddRecord("CLOSE_AUCTION", map[string]any{
		"painting_id": req.PaintingID, "winner": lastBidder, "final_price": lastAmount,
	})
	respondJSON(w, 200, map[string]string{"status": "ok"})
}

func handleGetChain(w http.ResponseWriter, r *http.Request) {
	chain.mu.Lock()
	data, _ := os.ReadFile(chain.File)
	var storage chainStorage
	json.Unmarshal(data, &storage)
	chain.Records = storage.Records
	chain.mu.Unlock()
	respondJSON(w, 200, chain.Records)
}

func handleValidate(w http.ResponseWriter, r *http.Request) {
	valid, err := chain.ReloadAndValidate()
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, 200, map[string]bool{"valid": valid})
}
