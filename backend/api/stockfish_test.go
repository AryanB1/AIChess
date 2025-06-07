package api_test

import (
	"AIChess/api"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockStockfishServer is a helper to create a mock Stockfish API server.
func mockStockfishServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler) // Corrected: treturn to return
}

func TestStockFishEndpoint_ValidRequest(t *testing.T) {
	server := mockStockfishServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("fen") != "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" {
			t.Error("Unexpected FEN received by mock Stockfish")
		}
		if r.URL.Query().Get("depth") != "10" {
			t.Error("Unexpected depth received by mock Stockfish")
		}
		w.Header().Set("Content-Type", "application/json")
		response := api.FullStockfishResponse{
			Success:  true,
			BestMove: "bestmove e2e4 continuation d1d2", // Ensure "bestmove " prefix
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	originalURL := api.StockfishAPIURL
	api.StockfishAPIURL = server.URL + "?" // The SendRequestToStockFish will append params
	defer func() { api.StockfishAPIURL = originalURL }()

	requestBody := []byte(`{"depth": 10, "fen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"}`)

	recorder := httptest.NewRecorder()

	req, err := http.NewRequest(http.MethodPost, "/engine-move", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}

	api.StockFishEndpoint(recorder, req)

	// Check for expected status code
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	// Check for Content-Type header
	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type header 'application/json', got %s", contentType)
	}

	// Parse the response body
	var response api.ParsedResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response body: %v", err)
	}

	// Check if success is true
	if !response.Success {
		t.Errorf("Expected success to be true, got %v", response.Success)
	}

	// Check if best move is not empty and correctly parsed
	if response.BestMove != "e2e4" { // Ensure ExtractBestMove works as expected
		t.Errorf("Expected best move to be 'e2e4', got %s", response.BestMove)
	}
}

func TestStockFishEndpoint_InvalidMethod(t *testing.T) {
	requestBody := []byte(`{"depth": 10, "fen": "startpos"}`)

	recorder := httptest.NewRecorder()

	// Create a request with GET method
	req, err := http.NewRequest(http.MethodGet, "/engine-move", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}

	api.StockFishEndpoint(recorder, req)

	// Check for expected status code
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}
}

// Renamed from TestStockFishEndpoint_InvalidBody to TestStockFishEndpoint_InvalidJSONBody for clarity
func TestStockFishEndpoint_InvalidJSONBody(t *testing.T) {
	requestBody := []byte(`{"depth": 10, "fen": "startpos"`) // Invalid JSON, missing closing brace

	recorder := httptest.NewRecorder()

	req, err := http.NewRequest(http.MethodPost, "/engine-move", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}

	api.StockFishEndpoint(recorder, req)

	// Check for expected status code
	if recorder.Code != http.StatusBadRequest { // Should be BadRequest for malformed JSON
		t.Errorf("Expected status code %d for invalid JSON, got %d. Body: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
}

func TestStockFishEndpoint_StockfishAPIFailure(t *testing.T) {
	server := mockStockfishServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // Simulate Stockfish API returning an error
	})
	defer server.Close()

	originalURL := api.StockfishAPIURL
	api.StockfishAPIURL = server.URL + "?"

	defer func() { api.StockfishAPIURL = originalURL }()

	requestBody := []byte(`{"depth": 10, "fen": "startpos"}`)
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/engine-move", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}

	api.StockFishEndpoint(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d on Stockfish API failure, got %d. Body: %s", http.StatusInternalServerError, recorder.Code, recorder.Body.String())
	}
}

func TestStockFishEndpoint_StockfishNonSuccessResponse(t *testing.T) {
	server := mockStockfishServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Simulate Stockfish API returning success:false
		response := api.FullStockfishResponse{Success: false, BestMove: ""}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	originalURL := api.StockfishAPIURL
	api.StockfishAPIURL = server.URL + "?"

	defer func() { api.StockfishAPIURL = originalURL }()

	requestBody := []byte(`{"depth": 10, "fen": "startpos"}`)
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/engine-move", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}

	api.StockFishEndpoint(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d on Stockfish non-success response, got %d. Body: %s", http.StatusInternalServerError, recorder.Code, recorder.Body.String())
	}
}

func TestStockFishEndpoint_OptionsRequest(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodOptions, "/engine-move", nil)
	if err != nil {
		t.Fatal(err)
	}
	api.StockFishEndpoint(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status OK for OPTIONS request, got %d", recorder.Code)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Error("CORS Access-Control-Allow-Origin not set correctly for OPTIONS request")
	}
	// Add more checks for other CORS headers if necessary
}

// Tests for StockfishHintEndpoint
func TestStockfishHintEndpoint_ValidRequest(t *testing.T) {
	server := mockStockfishServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("fen") != "r1bqkbnr/pp1ppppp/2n5/2p5/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3" {
			t.Error("Hint: Unexpected FEN received by mock Stockfish")
		}
		if r.URL.Query().Get("depth") != "8" {
			t.Error("Hint: Unexpected depth received by mock Stockfish")
		}
		w.Header().Set("Content-Type", "application/json")
		response := api.FullStockfishResponse{
			Success:  true,
			BestMove: "bestmove g1f3 continuation ...", // Ensure "bestmove " prefix
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	originalURL := api.StockfishAPIURL
	api.StockfishAPIURL = server.URL + "?"

	defer func() { api.StockfishAPIURL = originalURL }()

	requestBody := []byte(`{"depth": 8, "fen": "r1bqkbnr/pp1ppppp/2n5/2p5/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3"}`)
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/get-hint", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}

	api.StockfishHintEndpoint(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Hint: Expected status code %d, got %d. Body: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	var hintResponse struct {
		Hint string `json:"hint"`
	}
	err = json.Unmarshal(recorder.Body.Bytes(), &hintResponse)
	if err != nil {
		t.Fatalf("Hint: Failed to unmarshal response: %v", err)
	}
	if hintResponse.Hint != "g1f3" {
		t.Errorf("Hint: Expected hint 'g1f3', got '%s'", hintResponse.Hint)
	}
}

func TestStockfishHintEndpoint_StockfishAPIFailure(t *testing.T) {
	server := mockStockfishServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // Simulate Stockfish API error
	})
	defer server.Close()

	originalURL := api.StockfishAPIURL
	api.StockfishAPIURL = server.URL + "?"

	defer func() { api.StockfishAPIURL = originalURL }()

	requestBody := []byte(`{"depth": 5, "fen": "startpos"}`)
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/get-hint", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}

	api.StockfishHintEndpoint(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Hint: Expected status %d on Stockfish API failure, got %d. Body: %s", http.StatusInternalServerError, recorder.Code, recorder.Body.String())
	}
}

func TestStockfishHintEndpoint_InvalidJSON(t *testing.T) {
	requestBody := []byte(`{"depth": 5, "fen": "startpos"`) // Malformed JSON
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/get-hint", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}
	api.StockfishHintEndpoint(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Hint: Expected status %d for invalid JSON, got %d. Body: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
}

func TestStockfishHintEndpoint_OptionsRequest(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodOptions, "/get-hint", nil)
	if err != nil {
		t.Fatal(err)
	}
	api.StockfishHintEndpoint(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Errorf("Hint: Expected status OK for OPTIONS, got %d", recorder.Code)
	}
}

// Tests for utility functions in stockfish.go
func TestExtractBestMove(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Valid full bestmove string", "bestmove e2e4 continuation d2d4", "e2e4"},
		{"Valid bestmove string no continuation", "bestmove e7e5", "e7e5"},
		{"String with only bestmove token", "bestmove", ""}, // Corrected expectation
		{"String with token and no move", "bestmove ", ""},  // Corrected expectation
		{"Empty string", "", ""},
		{"Random string not starting with bestmove", "e2e4", ""},
		{"No bestmove token", "some other string e2e4", ""},
		{"Bestmove token not at start", "move bestmove e2e4", ""},
		{"Bestmove with numbers and letters", "bestmove a1b2", "a1b2"},
		{"Bestmove with promotion", "bestmove e7e8q", "e7e8q"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := api.ExtractBestMove(tt.input)
			if actual != tt.expected {
				t.Errorf("ExtractBestMove(\"%s\"): expected \"%s\", actual \"%s\"", tt.input, tt.expected, actual)
			}
		})
	}
}

func TestSendRequestToStockFish_Success(t *testing.T) {
	expectedFen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	expectedDepth := 15
	server := mockStockfishServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request to Stockfish, got %s", r.Method)
		}
		fen := r.URL.Query().Get("fen")
		depth := r.URL.Query().Get("depth")
		if fen != expectedFen {
			t.Errorf("Stockfish mock: Expected FEN %s, got %s", expectedFen, fen)
		}
		if depth != fmt.Sprintf("%d", expectedDepth) {
			t.Errorf("Stockfish mock: Expected depth %d, got %s", expectedDepth, depth)
		}
		w.Header().Set("Content-Type", "application/json")
		response := api.FullStockfishResponse{Success: true, BestMove: "bestmove d2d4"}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	originalURL := api.StockfishAPIURL
	api.StockfishAPIURL = server.URL + "?" // The SendRequestToStockFish adds params
	defer func() { api.StockfishAPIURL = originalURL }()

	params := api.GetStockfishParams{Fen: expectedFen, Depth: expectedDepth}
	resp, err := api.SendRequestToStockFish(params)
	if err != nil {
		t.Fatalf("SendRequestToStockFish failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK from Stockfish, got %s", resp.Status)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	var stockfishResponse api.FullStockfishResponse
	if err := json.Unmarshal(bodyBytes, &stockfishResponse); err != nil {
		t.Fatalf("Failed to unmarshal Stockfish response: %v", err)
	}
	if !stockfishResponse.Success {
		t.Error("Expected Stockfish response success to be true")
	}
}

func TestSendRequestToStockFish_APIReturnsError(t *testing.T) {
	server := mockStockfishServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable) // Simulate an API error
	})
	defer server.Close()

	originalURL := api.StockfishAPIURL
	api.StockfishAPIURL = server.URL + "?"

	defer func() { api.StockfishAPIURL = originalURL }()

	params := api.GetStockfishParams{Fen: "startpos", Depth: 10}
	resp, err := api.SendRequestToStockFish(params)
	if err != nil {
		// This is an error in creating or sending the request itself, not an API error response
		t.Fatalf("SendRequestToStockFish failed unexpectedly: %v", err)
	}
	defer resp.Body.Close()

	// We expect to receive the error status code from the server
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d from Stockfish, got %s", http.StatusServiceUnavailable, resp.Status)
	}
}

func TestSendRequestToStockFish_InvalidURL(t *testing.T) {
	originalURL := api.StockfishAPIURL
	// Provide an invalid URL that http.NewRequest would fail on
	api.StockfishAPIURL = "http://invalid url with spaces.com/api?"
	defer func() { api.StockfishAPIURL = originalURL }()

	params := api.GetStockfishParams{Fen: "startpos", Depth: 10}
	_, err := api.SendRequestToStockFish(params)
	if err == nil {
		t.Fatal("Expected error for invalid URL, got nil")
	}
	// Check if the error is a URL parsing error
	if !strings.Contains(err.Error(), "invalid control character in URL") && !strings.Contains(err.Error(), "parse") {
		t.Errorf("Expected URL parsing error, got: %v", err)
	}
}

func TestSendRequestToStockFish_ClientDoError(t *testing.T) {
	originalURL := api.StockfishAPIURL
	// Point to a non-resolvable or non-listening address to simulate network error
	api.StockfishAPIURL = "http://localhost:12345/nonexistent?" // Use a port that is unlikely to be in use
	defer func() { api.StockfishAPIURL = originalURL }()

	params := api.GetStockfishParams{Fen: "startpos", Depth: 10}
	_, err := api.SendRequestToStockFish(params)
	if err == nil {
		t.Fatal("Expected error from client.Do, got nil")
	}
	// The error message can vary, so we check for common connection refused messages.
	if !strings.Contains(err.Error(), "connect: connection refused") && !strings.Contains(err.Error(), "no such host") {
		t.Logf("Note: Client.Do error might vary based on OS/network. Got: %v", err) // Log for informational purposes
	}
}
