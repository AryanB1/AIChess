package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type GetStockfishParams struct {
	Depth int    `json:"depth"`
	Fen   string `json:"fen"`
}

func main() {
	fmt.Println("Hello, World!")
	http.HandleFunc("/", handler)
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		return
	}
}
func handler(w http.ResponseWriter, r *http.Request) {
	// Call sendRequest to get the response
	params := GetStockfishParams{
		Depth: 15,
		Fen:   "rn1q1rk1/pp2b1pp/2p2n2/3p1pB1/3P4/1QP2N2/PP1N1PPP/R4RK1 b - - 1 11",
	}
	resp, err := sendRequest(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check for successful response status code
	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Error from Stockfish API: %s", resp.Status), resp.StatusCode)
		return
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close() // Ensure the body is closed

	// Set the content type to application/json
	w.Header().Set("Content-Type", "application/json")

	// Write the response body to the response writer
	_, err = w.Write(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func sendRequest(body GetStockfishParams) (*http.Response, error) {
	params := url.Values{}
	params.Add("fen", body.Fen)
	params.Add("depth", fmt.Sprintf("%d", body.Depth)) // Add depth as a query parameter

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
