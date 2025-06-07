package api_test

import (
	"AIChess/api"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockGeminiServer is a helper to create a mock Gemini API server.
func mockGeminiServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestGetGeminiExplanation_Success(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	bestMove := "e2e4"
	expectedExplanation := "This is a great move because it opens lines for the queen and bishop."

	server := mockGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request to Gemini, got %s", r.Method)
		}
		var payload api.GeminiRequestPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		// Basic check for FEN and bestMove in the prompt
		if !strings.Contains(payload.Contents[0].Parts[0].Text, fen) {
			t.Errorf("Prompt does not contain FEN. Got: %s", payload.Contents[0].Parts[0].Text)
		}
		if !strings.Contains(payload.Contents[0].Parts[0].Text, bestMove) {
			t.Errorf("Prompt does not contain bestMove. Got: %s", payload.Contents[0].Parts[0].Text)
		}

		w.Header().Set("Content-Type", "application/json")
		response := api.GeminiResponsePayload{
			Candidates: []api.Candidate{
				{
					Content: api.Content{
						Parts: []api.Part{{Text: expectedExplanation}},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	// Set up environment for the test
	originalAPIKey := os.Getenv("GEMINI_API_KEY")
	os.Setenv("GEMINI_API_KEY", "test_api_key_for_gemini")
	defer os.Setenv("GEMINI_API_KEY", originalAPIKey)

	originalGetAPIURL := api.GetGeminiAPIURL
	api.GetGeminiAPIURL = func(apiKey string) string { return server.URL } // Override with mock server URL
	defer func() { api.GetGeminiAPIURL = originalGetAPIURL }()             // Restore original

	explanation, err := api.GetGeminiExplanation(fen, bestMove)
	if err != nil {
		t.Fatalf("GetGeminiExplanation failed: %v", err)
	}

	if explanation != expectedExplanation {
		t.Errorf("Expected explanation \"%s\", got \"%s\"", expectedExplanation, explanation)
	}
}

func TestGetGeminiExplanation_NoAPIKey(t *testing.T) {
	originalAPIKey := os.Getenv("GEMINI_API_KEY")
	os.Unsetenv("GEMINI_API_KEY") // Ensure API key is not set
	defer os.Setenv("GEMINI_API_KEY", originalAPIKey)

	_, err := api.GetGeminiExplanation("fen", "move")
	if err == nil {
		t.Fatal("Expected error when GEMINI_API_KEY is not set, got nil")
	}
	if !strings.Contains(err.Error(), "GEMINI_API_KEY not set") {
		t.Errorf("Expected error message about API key, got: %v", err)
	}
}

func TestGetGeminiExplanation_APIError(t *testing.T) {
	server := mockGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Gemini internal error"))
	})
	defer server.Close()

	originalAPIKey := os.Getenv("GEMINI_API_KEY")
	os.Setenv("GEMINI_API_KEY", "test_key")
	defer os.Setenv("GEMINI_API_KEY", originalAPIKey)

	originalGetAPIURL := api.GetGeminiAPIURL
	api.GetGeminiAPIURL = func(apiKey string) string { return server.URL }
	defer func() { api.GetGeminiAPIURL = originalGetAPIURL }()

	_, err := api.GetGeminiExplanation("fen", "move")
	if err == nil {
		t.Fatal("Expected error from Gemini API, got nil")
	}
	if !strings.Contains(err.Error(), "gemini API request failed with status 500 Internal Server Error: Gemini internal error") {
		t.Errorf("Expected error message about API failure, got: %v", err)
	}
}

func TestGetGeminiExplanation_NoCandidates(t *testing.T) {
	server := mockGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := api.GeminiResponsePayload{Candidates: []api.Candidate{}}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	originalAPIKey := os.Getenv("GEMINI_API_KEY")
	os.Setenv("GEMINI_API_KEY", "test_key")
	defer os.Setenv("GEMINI_API_KEY", originalAPIKey)

	originalGetAPIURL := api.GetGeminiAPIURL
	api.GetGeminiAPIURL = func(apiKey string) string { return server.URL }
	defer func() { api.GetGeminiAPIURL = originalGetAPIURL }()

	_, err := api.GetGeminiExplanation("fen", "move")
	if err == nil {
		t.Fatal("Expected error when no candidates in response, got nil")
	}
	if !strings.Contains(err.Error(), "no explanation found in Gemini API response") {
		t.Errorf("Expected error message about no explanation, got: %v", err)
	}
}

func TestGetGeminiExplanation_ContentBlocked(t *testing.T) {
	server := mockGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := api.GeminiResponsePayload{
			Candidates: []api.Candidate{},
			PromptFeedback: &api.PromptFeedback{
				SafetyRatings: []api.SafetyRating{
					{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Probability: "HIGH"},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	originalAPIKey := os.Getenv("GEMINI_API_KEY")
	os.Setenv("GEMINI_API_KEY", "test_key")
	defer os.Setenv("GEMINI_API_KEY", originalAPIKey)

	originalGetAPIURL := api.GetGeminiAPIURL
	api.GetGeminiAPIURL = func(apiKey string) string { return server.URL }
	defer func() { api.GetGeminiAPIURL = originalGetAPIURL }()

	_, err := api.GetGeminiExplanation("fen", "move")
	if err == nil {
		t.Fatal("Expected error when content is blocked, got nil")
	}
	fmt.Println(err.Error())
	if !strings.Contains(err.Error(), "content blocked by Gemini API due to safety concerns") {
		t.Errorf("Expected error message about content blocking, got: %v", err)
	}
}

// Test loadEnvFromFile (optional, as it's an internal helper, but good for completeness)
func TestLoadEnvFromFile_Success(t *testing.T) {
	// Create a temporary .env file
	content := []byte("TEST_KEY_GEMINI=test_value_gemini\n# This is a comment\n ANOTHER_KEY = another_value ")
	tmpfile, err := os.CreateTemp("", ".env")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	wd, _ := os.Getwd()
	originalDotEnvPath := api.DotEnvPath // Assuming you add this to api/gemini.go
	api.DotEnvPath = filepath.Join(wd, ".env.test_gemini_load")
	defer func() {
		os.Remove(api.DotEnvPath) // Clean up our test .env
		api.DotEnvPath = originalDotEnvPath
	}()

	testEnvContent := "GEMINI_API_KEY=loaded_from_test_dotenv"
	if err := os.WriteFile(api.DotEnvPath, []byte(testEnvContent), 0644); err != nil {
		t.Fatalf("Failed to write test .env file: %v", err)
	}

	// Unset the key first to ensure it's loaded from the file
	originalEnvVal := os.Getenv("GEMINI_API_KEY")
	os.Unsetenv("GEMINI_API_KEY")
	defer os.Setenv("GEMINI_API_KEY", originalEnvVal)

	// Mock the server so GetGeminiExplanation doesn't fail due to network
	server := mockGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := api.GeminiResponsePayload{Candidates: []api.Candidate{{Content: api.Content{Parts: []api.Part{{Text: "ok"}}}}}}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()
	originalGetAPIURL := api.GetGeminiAPIURL
	api.GetGeminiAPIURL = func(apiKey string) string { return server.URL }
	defer func() { api.GetGeminiAPIURL = originalGetAPIURL }()

	// This call will trigger loadEnvOnce.Do(loadEnvFromFile)
	_, _ = api.GetGeminiExplanation("fen", "move")

	if os.Getenv("GEMINI_API_KEY") != "loaded_from_test_dotenv" {
		t.Errorf("Expected GEMINI_API_KEY to be loaded from .env file, got '%s'", os.Getenv("GEMINI_API_KEY"))
	}
}

func TestLoadEnvFromFile_FileNotFound(t *testing.T) {
	// Ensure no .env file exists at the path loadEnvFromFile checks
	originalDotEnvPath := api.DotEnvPath
	api.DotEnvPath = "./non_existent_test.env"
	defer func() { api.DotEnvPath = originalDotEnvPath }()

	keyToTest := "SOME_KEY_THAT_SHOULD_NOT_BE_SET_BY_MISSING_DOTENV"
	originalVal := os.Getenv(keyToTest)
	os.Unsetenv(keyToTest)
	defer os.Setenv(keyToTest, originalVal)

	// Reset loadEnvOnce so loadEnvFromFile actually runs again
	api.ResetLoadEnvOnce() // You'll need to add this function to api/gemini.go

	// Call a function that would trigger it
	originalAPIKey := os.Getenv("GEMINI_API_KEY")
	os.Setenv("GEMINI_API_KEY", "temp_for_call") // Set it so GetGeminiExplanation doesn't fail early
	defer os.Setenv("GEMINI_API_KEY", originalAPIKey)

	server := mockGeminiServer(t, func(w http.ResponseWriter, r *http.Request) { /* ... */ })
	defer server.Close()
	originalGetAPIURL := api.GetGeminiAPIURL
	api.GetGeminiAPIURL = func(apiKey string) string { return server.URL }
	defer func() { api.GetGeminiAPIURL = originalGetAPIURL }()

	_, _ = api.GetGeminiExplanation("fen", "move")

	if os.Getenv(keyToTest) != "" {
		t.Errorf("Expected %s to be empty after trying to load non-existent .env, but got '%s'", keyToTest, os.Getenv(keyToTest))
	}
}
