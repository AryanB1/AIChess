package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// request struct
type GetStockfishParams struct {
	Depth int    `json:"depth"`
	Fen   string `json:"fen"`
}

// response structs
type FullStockfishResponse struct {
	Success      bool    `json:"success"`
	Evaluation   float64 `json:"evaluation"`
	Mate         *int    `json:"mate"`
	BestMove     string  `json:"bestmove"`
	Continuation string  `json:"continuation"`
}

type ParsedResponse struct {
	Success  bool   `json:"success"`
	BestMove string `json:"bestmove"`
}

var StockfishAPIURL = "https://stockfish.online/api/s/v2.php?"

// Endpoint that frontend calls to get stockfish moves
func StockFishEndpoint(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	w.Header().Set("Content-Type", "application/json")
	// Validates that the method is POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Reads and validates request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parses JSON body into GetStockfishParams struct
	var params GetStockfishParams
	err = json.Unmarshal(body, &params)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Sends request to stockfish
	resp, err := SendRequestToStockFish(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Reads the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Unmarshals the response into FullStockfishResponse struct
	var fullResponse FullStockfishResponse
	err = json.Unmarshal(respBody, &fullResponse)
	if err != nil { // Simplified error check, was: err != nil || !fullResponse.Success
		http.Error(w, "Failed to parse Stockfish API response: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !fullResponse.Success { // Added separate check for Success field after unmarshalling
		http.Error(w, "Stockfish API call was not successful based on response data", http.StatusInternalServerError)
		return
	}
	// Extracts necessary data
	parsedResponse := ParsedResponse{
		Success:  fullResponse.Success,
		BestMove: ExtractBestMove(fullResponse.BestMove),
	}

	// Sets and writes response
	w.Header().Set("Content-Type", "application/json")
	jsonResponse, err := json.Marshal(parsedResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(jsonResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func StockfishHintEndpoint(w http.ResponseWriter, r *http.Request) {
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

	var params GetStockfishParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid JSON format: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Sends request to stockfish
	resp, err := SendRequestToStockFish(params)
	if err != nil {
		http.Error(w, "Failed to call Stockfish API: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read Stockfish API response body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var fullResponse FullStockfishResponse
	if err := json.Unmarshal(respBody, &fullResponse); err != nil {
		http.Error(w, "Failed to parse Stockfish API response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !fullResponse.Success {
		http.Error(w, "Stockfish API call was not successful", http.StatusInternalServerError)
		return
	}

	hintResponse := struct {
		Hint string `json:"hint"`
	}{
		Hint: ExtractBestMove(fullResponse.BestMove),
	}

	if err := json.NewEncoder(w).Encode(hintResponse); err != nil {
		http.Error(w, "Failed to write hint response: "+err.Error(), http.StatusInternalServerError)
	}
}

func SendRequestToStockFish(body GetStockfishParams) (*http.Response, error) {
	params := url.Values{}
	params.Add("fen", body.Fen)
	params.Add("depth", fmt.Sprintf("%d", body.Depth))

	encodedParams := params.Encode()

	req, err := http.NewRequest("GET", StockfishAPIURL+encodedParams, nil)
	if err != nil {
		return nil, err
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func ExtractBestMove(bestMoveInput string) string { // Renamed parameter to avoid conflict
	parts := strings.Split(bestMoveInput, " ")
	// Expects format like "bestmove e2e4 continuation ..."
	if len(parts) >= 2 && parts[0] == "bestmove" {
		return parts[1]
	}
	return ""
}
