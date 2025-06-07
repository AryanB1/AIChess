package main

import (
	"AIChess/api"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Mock Stockfish server
func mockStockfishServer(t *testing.T, expectedFen string, expectedDepth int, response api.FullStockfishResponse, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/s/v2.php" {
			t.Errorf("Expected to request '/api/s/v2.php', got '%s'", r.URL.Path)
		}
		fen := r.URL.Query().Get("fen")
		depthStr := r.URL.Query().Get("depth")

		if fen != expectedFen {
			t.Errorf("Expected fen '%s', got '%s'", expectedFen, fen)
		}
		// Convert depthStr to int for comparison if needed, here we just check if it's passed
		if depthStr == "" { // Simplified check
			t.Error("Expected depth parameter, got none")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
	}))
}

// Mock Gemini server
func mockGeminiServer(t *testing.T, expectedFen string, expectedMove string, explanation string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload api.GeminiRequestPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, "Failed to decode payload", http.StatusBadRequest)
			return
		}

		expectedPromptPart := "Given the following chess position in FEN notation: " + expectedFen
		if !strings.Contains(payload.Contents[0].Parts[0].Text, expectedPromptPart) {
			t.Errorf("Gemini prompt does not contain expected FEN. Got: %s", payload.Contents[0].Parts[0].Text)
		}
		expectedMovePart := "Stockfish suggests the best move is " + expectedMove
		if !strings.Contains(payload.Contents[0].Parts[0].Text, expectedMovePart) {
			t.Errorf("Gemini prompt does not contain expected best move. Got: %s", payload.Contents[0].Parts[0].Text)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		response := api.GeminiResponsePayload{
			Candidates: []api.Candidate{
				{
					Content: api.Content{
						Parts: []api.Part{
							{Text: explanation},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
}

func TestGetAIExplanationEndpoint_ValidRequest(t *testing.T) {
	initialFen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	stockfishBestMove := "e2e4" // Stockfish's best move (actual UCI, not just the first part)
	extractedBestMove := "e2e4" // The part we expect to be extracted and sent to Gemini
	geminiExplanation := "Moving the king's pawn forward is a strong opening."

	// Mock Stockfish
	stockfishServer := mockStockfishServer(t, initialFen, 12, api.FullStockfishResponse{
		Success:  true,
		BestMove: "bestmove " + stockfishBestMove + " continuation ...", // Ensure "bestmove " prefix
	}, http.StatusOK)
	defer stockfishServer.Close()

	// Override Stockfish API URL to use mock server
	originalStockfishURL := api.StockfishAPIURL                  // Assume StockfishAPIURL is exported or settable
	api.StockfishAPIURL = stockfishServer.URL + "/api/s/v2.php?" // Adjust if your SendRequestToStockFish constructs URL differently
	defer func() { api.StockfishAPIURL = originalStockfishURL }()

	// Mock Gemini
	geminiServer := mockGeminiServer(t, initialFen, extractedBestMove, geminiExplanation, http.StatusOK)
	defer geminiServer.Close()

	// Set Gemini API Key and URL
	os.Setenv("GEMINI_API_KEY", "test_api_key")
	originalGeminiURLFunc := api.GetGeminiAPIURL                                 // Store original function
	api.GetGeminiAPIURL = func(apiKey string) string { return geminiServer.URL } // Override with mock
	defer func() { api.GetGeminiAPIURL = originalGeminiURLFunc }()               // Restore original function

	requestBody := map[string]string{"fen": initialFen}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest(http.MethodPost, "/get-ai-explanation", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(GetAIExplanationEndpoint)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status OK; got %v. Body: %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Explanation string `json:"explanation"`
	}
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Explanation != geminiExplanation {
		t.Errorf("Expected explanation '%s'; got '%s'", geminiExplanation, response.Explanation)
	}
}

func TestGetAIExplanationEndpoint_InvalidMethod(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/get-ai-explanation", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(GetAIExplanationEndpoint)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status MethodNotAllowed; got %v", recorder.Code)
	}
}

func TestGetAIExplanationEndpoint_EmptyFEN(t *testing.T) {
	requestBody := map[string]string{"fen": ""}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest(http.MethodPost, "/get-ai-explanation", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(GetAIExplanationEndpoint)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest; got %v. Body: %s", recorder.Code, recorder.Body.String())
	}
}

func TestGetAIExplanationEndpoint_StockfishFailure(t *testing.T) {
	initialFen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	stockfishServer := mockStockfishServer(t, initialFen, 12, api.FullStockfishResponse{}, http.StatusInternalServerError)
	defer stockfishServer.Close()
	originalStockfishURL := api.StockfishAPIURL
	api.StockfishAPIURL = stockfishServer.URL + "/api/s/v2.php?"
	defer func() { api.StockfishAPIURL = originalStockfishURL }()

	requestBody := map[string]string{"fen": initialFen}
	jsonBody, _ := json.Marshal(requestBody)
	req, err := http.NewRequest(http.MethodPost, "/get-ai-explanation", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(GetAIExplanationEndpoint)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status InternalServerError on Stockfish failure; got %v. Body: %s", recorder.Code, recorder.Body.String())
	}
}

func TestGetAIExplanationEndpoint_StockfishNonSuccessResponse(t *testing.T) {
	initialFen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	stockfishServer := mockStockfishServer(t, initialFen, 12, api.FullStockfishResponse{Success: false}, http.StatusOK)
	defer stockfishServer.Close()
	originalStockfishURL := api.StockfishAPIURL
	api.StockfishAPIURL = stockfishServer.URL + "/api/s/v2.php?"
	defer func() { api.StockfishAPIURL = originalStockfishURL }()

	requestBody := map[string]string{"fen": initialFen}
	jsonBody, _ := json.Marshal(requestBody)
	req, err := http.NewRequest(http.MethodPost, "/get-ai-explanation", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(GetAIExplanationEndpoint)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status InternalServerError on Stockfish non-success; got %v. Body: %s", recorder.Code, recorder.Body.String())
	}
}

func TestGetAIExplanationEndpoint_GeminiFailure(t *testing.T) {
	initialFen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	stockfishBestMove := "e2e4"

	stockfishServer := mockStockfishServer(t, initialFen, 12, api.FullStockfishResponse{
		Success:  true,
		BestMove: "bestmove " + stockfishBestMove,
	}, http.StatusOK)
	defer stockfishServer.Close()
	originalStockfishURL := api.StockfishAPIURL
	api.StockfishAPIURL = stockfishServer.URL + "/api/s/v2.php?"
	api.StockfishAPIURL = stockfishServer.URL + "/api/s/v2.php?"
	defer func() { api.StockfishAPIURL = originalStockfishURL }()

	geminiServer := mockGeminiServer(t, initialFen, stockfishBestMove, "", http.StatusInternalServerError)
	defer geminiServer.Close()
	originalGeminiURLFunc := api.GetGeminiAPIURL
	api.GetGeminiAPIURL = func(apiKey string) string { return geminiServer.URL }
	defer func() { api.GetGeminiAPIURL = originalGeminiURLFunc }()

	requestBody := map[string]string{"fen": initialFen}
	jsonBody, _ := json.Marshal(requestBody)
	req, err := http.NewRequest(http.MethodPost, "/get-ai-explanation", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(GetAIExplanationEndpoint)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status InternalServerError on Gemini failure; got %v. Body: %s", recorder.Code, recorder.Body.String())
	}
}

func TestGetAIExplanationEndpoint_GeminiNoExplanation(t *testing.T) {
	initialFen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	stockfishBestMove := "e2e4"

	stockfishServer := mockStockfishServer(t, initialFen, 12, api.FullStockfishResponse{
		Success:  true,
		BestMove: "bestmove " + stockfishBestMove,
	}, http.StatusOK)
	defer stockfishServer.Close()
	originalStockfishURL := api.StockfishAPIURL
	api.StockfishAPIURL = stockfishServer.URL + "/api/s/v2.php?"
	defer func() { api.StockfishAPIURL = originalStockfishURL }()

	// Mock Gemini to return success but no candidates/explanation
	geminiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)           // Gemini request itself is OK
		response := api.GeminiResponsePayload{ // But payload has no explanation
			Candidates: []api.Candidate{},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer geminiServer.Close()
	os.Setenv("GEMINI_API_KEY", "test_api_key")
	originalGeminiURLFunc := api.GetGeminiAPIURL
	api.GetGeminiAPIURL = func(apiKey string) string { return geminiServer.URL }
	defer func() { api.GetGeminiAPIURL = originalGeminiURLFunc }()

	requestBody := map[string]string{"fen": initialFen}
	jsonBody, _ := json.Marshal(requestBody)
	req, err := http.NewRequest(http.MethodPost, "/get-ai-explanation", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(GetAIExplanationEndpoint)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status InternalServerError when Gemini returns no explanation; got %v. Body: %s", recorder.Code, recorder.Body.String())
	}
}

func TestGetAIExplanationEndpoint_OptionsRequest(t *testing.T) {
	req, err := http.NewRequest(http.MethodOptions, "/get-ai-explanation", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(GetAIExplanationEndpoint)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status OK for OPTIONS request; got %v", recorder.Code)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("Expected Access-Control-Allow-Origin to be http://localhost:3000, got %s", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
	if recorder.Header().Get("Access-Control-Allow-Methods") != "POST, OPTIONS" {
		t.Errorf("Expected Access-Control-Allow-Methods to be POST, OPTIONS, got %s", recorder.Header().Get("Access-Control-Allow-Methods"))
	}
	if recorder.Header().Get("Access-Control-Allow-Headers") != "Content-Type" {
		t.Errorf("Expected Access-Control-Allow-Headers to be Content-Type, got %s", recorder.Header().Get("Access-Control-Allow-Headers"))
	}
}

func TestGetAIExplanationEndpoint_InvalidJSONBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/get-ai-explanation", bytes.NewBufferString("{not_a_json"))
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(GetAIExplanationEndpoint)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest for invalid JSON; got %v. Body: %s", recorder.Code, recorder.Body.String())
	}
}

func TestGetAIExplanationEndpoint_StockfishBadBestMoveFormat(t *testing.T) {
	initialFen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	// Stockfish returns a bestmove string that doesn't conform to "bestmove <move> ..."
	stockfishServer := mockStockfishServer(t, initialFen, 12, api.FullStockfishResponse{
		Success:  true,
		BestMove: "invalidformat", // Missing "bestmove " prefix and space
	}, http.StatusOK)
	defer stockfishServer.Close()

	originalStockfishURL := api.StockfishAPIURL
	api.StockfishAPIURL = stockfishServer.URL + "/api/s/v2.php?"
	defer func() { api.StockfishAPIURL = originalStockfishURL }()

	os.Setenv("GEMINI_API_KEY", "test_api_key") // Still need Gemini key for later stages if reached

	requestBody := map[string]string{"fen": initialFen}
	jsonBody, _ := json.Marshal(requestBody)
	req, err := http.NewRequest(http.MethodPost, "/get-ai-explanation", bytes.NewBuffer(jsonBody))
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(GetAIExplanationEndpoint)
	handler.ServeHTTP(recorder, req)

	// This should result in an error because ExtractBestMove will return ""
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status InternalServerError for bad Stockfish bestmove format; got %v. Body: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "Could not extract best move") {
		t.Errorf("Expected error message about extracting best move, got: %s", recorder.Body.String())
	}
}
