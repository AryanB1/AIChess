package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type GetStockfishParams struct {
	Depth int    `json:"depth"`
	Fen   string `json:"fen"`
}

type FullStockfishResponse struct {
	Success      bool    `json:"success"`
	Evaluation   float64 `json:"evaluation"`
	Mate         *bool   `json:"mate"`
	BestMove     string  `json:"bestmove"`
	Continuation string  `json:"continuation"`
}

type ParsedResponse struct {
	Success  bool   `json:"success"`
	BestMove string `json:"bestmove"`
}

func main() {
	fmt.Println("Hello, World!")
	http.HandleFunc("/engine-move", handler)
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		return
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
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
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the body of the POST request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse the JSON body into GetStockfishParams struct
	var params GetStockfishParams
	err = json.Unmarshal(body, &params)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Call sendRequest to get the response
	resp, err := sendRequest(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Unmarshal the response into FullStockfishResponse struct
	var fullResponse FullStockfishResponse
	err = json.Unmarshal(respBody, &fullResponse)
	if err != nil {
		http.Error(w, "Failed to parse Stockfish API response", http.StatusInternalServerError)
		return
	}

	// Extract the necessary data
	parsedResponse := ParsedResponse{
		Success:  fullResponse.Success,
		BestMove: extractBestMove(fullResponse.BestMove),
	}

	// Set the content type to application/json
	w.Header().Set("Content-Type", "application/json")

	// Write the parsed JSON response
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

func sendRequest(body GetStockfishParams) (*http.Response, error) {
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

func extractBestMove(bestMove string) string {
	parts := strings.Split(bestMove, " ")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}
