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

// Endpoint that frontend calls to get stockfish moves
func StockFishEndpoint(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	// if r.Method == http.MethodOptions {
	// 	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	// 	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	// 	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	// 	w.WriteHeader(http.StatusOK)
	// 	return
	// }
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
	if err != nil || fullResponse.Success == false {
		http.Error(w, "Failed to parse Stockfish API response", http.StatusInternalServerError)
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

// Calls the stockfish api and gets the best moves
func SendRequestToStockFish(body GetStockfishParams) (*http.Response, error) {
	params := url.Values{}
	params.Add("fen", body.Fen)
	params.Add("depth", fmt.Sprintf("%d", body.Depth))

	encodedParams := params.Encode()

	req, err := http.NewRequest("GET", "https://stockfish.online/api/s/v2.php?"+encodedParams, nil)
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

// Parses stockfish response to get the formatted move that is then passed to frontend
func ExtractBestMove(bestMove string) string {
	parts := strings.Split(bestMove, " ")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}
