package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

var chain *AuctionChain

func main() {
	var err error
	chain, err = NewChain("chain_storage.json")
	if err != nil {
		log.Fatal("Ошибка инициализации цепи: ", err)
	}

	mux := http.NewServeMux()

	// API
	mux.HandleFunc("/api/chain", handleGetChain)
	mux.HandleFunc("/api/auction/create", handleCreateAuction)
	mux.HandleFunc("/api/auction/bid", handlePlaceBid)
	mux.HandleFunc("/api/auction/close", handleCloseAuction)
	mux.HandleFunc("/api/hack/simulate", handleSimulateHack)
	mux.HandleFunc("/api/validate", handleValidate)

	// Статика
	mux.Handle("/", http.FileServer(http.Dir("static")))

	fmt.Println("🖼  Art Auction Blockchain running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func respondJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func handleCreateAuction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
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

	fmt.Printf("📥 Создаю аукцион: ID=%s, Владелец=%s, Цена=%.2f\n", req.PaintingID, req.Owner, req.StartPrice)

	chain.AddRecord("CREATE_AUCTION", map[string]any{
		"painting_id": req.PaintingID,
		"title":       req.Title,
		"owner":       req.Owner,
		"start_price": req.StartPrice,
		"current_bid": req.StartPrice,
		"bidder":      req.Owner,
	})

	respondJSON(w, 200, map[string]string{"status": "ok"})
}

func handlePlaceBid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
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

	// 🛡 Проверка: новая ставка должна быть СТРОГО выше текущей
	currentPrice, isOpen := chain.GetCurrentPrice(req.PaintingID)
	if !isOpen {
		respondJSON(w, 400, map[string]string{"error": "Аукцион не найден или уже закрыт"})
		return
	}
	if req.Amount <= currentPrice {
		respondJSON(w, 400, map[string]string{
			"error": fmt.Sprintf("❌ Ставка отклонена: %.2f ≤ текущей цене (%.2f). Повысьте ставку!", req.Amount, currentPrice),
		})
		return
	}

	chain.AddRecord("PLACE_BID", map[string]any{
		"painting_id": req.PaintingID,
		"bidder":      req.Bidder,
		"amount":      req.Amount,
	})
	respondJSON(w, 200, map[string]string{"status": "ok"})
}

func handleCloseAuction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
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
		"painting_id": req.PaintingID,
		"winner":      lastBidder,
		"final_price": lastAmount,
	})
	respondJSON(w, 200, map[string]string{"status": "ok"})
}

func handleGetChain(w http.ResponseWriter, r *http.Request) {
	chain.mu.RLock()
	defer chain.mu.RUnlock()
	respondJSON(w, 200, chain.Records)
}

func handleValidate(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, 200, map[string]bool{"valid": chain.Validate()})
}

func handleSimulateHack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}
	var req struct {
		Index    int     `json:"index"`
		NewOwner string  `json:"new_owner"`
		NewPrice float64 `json:"new_price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}

	rec, reason, nextBroken := chain.SimulateHack(req.Index, map[string]any{
		"owner":       req.NewOwner,
		"current_bid": req.NewPrice,
	})
	respondJSON(w, 200, map[string]any{
		"modified_block": rec,
		"broken_reason":  reason,
		"next_broken":    nextBroken,
	})
}
