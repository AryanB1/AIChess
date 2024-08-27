package api_test

import (
	"AIChess/api"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStockFishEndpoint_ValidRequest(t *testing.T) {
	requestBody := []byte(`{"depth": 10, "fen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"}`)

	recorder := httptest.NewRecorder()

	req, err := http.NewRequest(http.MethodPost, "/engine-move", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}

	api.StockFishEndpoint(recorder, req)

	// Check for expected status code
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
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

	// Check if best move is not empty
	if response.BestMove == "" {
		t.Error("Expected best move to be non-empty")
	}
}

func TestStockFishEndpoint_InvalidMethod(t *testing.T) {
	requestBody := []byte(`{"depth": 10, "fen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"}`)

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

func TestStockFishEndpoint_InvalidBody(t *testing.T) {
	requestBody := []byte(`{"depth": 10}`)

	recorder := httptest.NewRecorder()

	// Create a request with POST method
	req, err := http.NewRequest(http.MethodPost, "/engine-move", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}

	api.StockFishEndpoint(recorder, req)

	// Check for expected status code
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
}
