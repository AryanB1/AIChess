package main

import (
	"AIChess/api"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/engine-move", api.StockFishEndpoint)
	http.HandleFunc("/get-hint", api.StockfishHintEndpoint)
	http.HandleFunc("/get-ai-explanation", GetAIExplanationEndpoint)

	port := "8081"

	fmt.Printf("Server starting on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func GetAIExplanationEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestBody struct {
		Fen string `json:"fen"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Failed to read request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if requestBody.Fen == "" {
		http.Error(w, "FEN string is required", http.StatusBadRequest)
		return
	}

	stockfishParams := api.GetStockfishParams{
		Fen:   requestBody.Fen,
		Depth: 12,
	}

	stockfishResp, err := api.SendRequestToStockFish(stockfishParams)
	if err != nil {
		http.Error(w, "Failed to get Stockfish move: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stockfishResp.Body.Close()

	if stockfishResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(stockfishResp.Body)
		http.Error(w, fmt.Sprintf("Stockfish API request failed with status %s: %s", stockfishResp.Status, string(bodyBytes)), http.StatusInternalServerError)
		return
	}

	var fullStockfishResponse api.FullStockfishResponse
	if err := json.NewDecoder(stockfishResp.Body).Decode(&fullStockfishResponse); err != nil {
		http.Error(w, "Failed to parse Stockfish API response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !fullStockfishResponse.Success || fullStockfishResponse.BestMove == "" {
		http.Error(w, "Stockfish did not return a successful move.", http.StatusInternalServerError)
		return
	}

	actualBestMove := api.ExtractBestMove(fullStockfishResponse.BestMove)
	if actualBestMove == "" {
		http.Error(w, "Could not extract best move from Stockfish response", http.StatusInternalServerError)
		return
	}

	explanation, err := api.GetGeminiExplanation(requestBody.Fen, actualBestMove)
	if err != nil {
		http.Error(w, "Failed to get Gemini explanation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		Explanation string `json:"explanation"`
	}{
		Explanation: explanation,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to write response: "+err.Error(), http.StatusInternalServerError)
	}
}
