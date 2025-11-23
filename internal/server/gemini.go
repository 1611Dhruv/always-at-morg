package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// ---------------------------------------------------------
// CONFIGURATION
// ---------------------------------------------------------

// TODO: PASTE YOUR REAL API KEY HERE
var GeminiAPIKey = "AIzaSyCxK1Ly7sHj56ZG35U_eelANT_JjMAcY60"

// Helper to get the URL, prioritizing Environment Variables for security
func getGeminiURL() string {
	key := GeminiAPIKey
	// Check if key is set in environment (Recommended)
	if envKey := os.Getenv("GEMINI_API_KEY"); envKey != "" {
		key = envKey
	}
	// UPDATED: Using "gemini-2.5-flash" based on your available models list
	return "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=" + key
}

// ---------------------------------------------------------
// STRUCTS
// ---------------------------------------------------------

type GeminiRiddle struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Hint     string `json:"hint"`
}

type MapClue struct {
	Location string `json:"location"`
	Riddle   string `json:"riddle"`
}

type TreasureMap struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Clues       []MapClue `json:"clues"`
}

type apiRequest struct {
	Contents []apiContent `json:"contents"`
}

type apiContent struct {
	Parts []apiPart `json:"parts"`
}

type apiPart struct {
	Text string `json:"text"`
}

type apiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// ---------------------------------------------------------
// FUNCTIONS
// ---------------------------------------------------------

func GenerateRiddle() (*GeminiRiddle, error) {
	// UPDATED PROMPT: Specifically asks for CS/Tech riddles
	prompt := `Generate a short, fun riddle about Computer Science, Programming, or Technology. 
	Return ONLY a JSON object with three fields: "question", "answer", and "hint". 
	Do not wrap in markdown code blocks.`

	jsonStr, err := rawGeminiCall(prompt)
	if err != nil {
		return nil, err
	}

	var riddle GeminiRiddle
	if err := json.Unmarshal([]byte(jsonStr), &riddle); err != nil {
		return nil, fmt.Errorf("failed to parse riddle JSON: %w", err)
	}

	return &riddle, nil
}

func GenerateTreasureMap(theme string) (*TreasureMap, error) {
	systemPrompt := `You are a creative treasure hunt generator. 
	Generate a fun and engaging treasure map with clues based on the theme provided.
	Return the response strictly as a JSON object.`
	
	userPrompt := fmt.Sprintf("%s. Theme: %s. Structure: { \"title\": \"...\", \"description\": \"...\", \"clues\": [ {\"location\": \"...\", \"riddle\": \"...\"} ] }", systemPrompt, theme)

	jsonStr, err := rawGeminiCall(userPrompt)
	if err != nil {
		return nil, err
	}

	var tMap TreasureMap
	if err := json.Unmarshal([]byte(jsonStr), &tMap); err != nil {
		return nil, fmt.Errorf("failed to parse map JSON: %w", err)
	}

	return &tMap, nil
}

// ---------------------------------------------------------
// HELPER
// ---------------------------------------------------------

func rawGeminiCall(prompt string) (string, error) {
	reqBody := apiRequest{
		Contents: []apiContent{
			{Parts: []apiPart{{Text: prompt}}},
		},
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Use the dynamic URL helper
	url := getGeminiURL()
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to connect to Gemini: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API Error %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp apiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("bad API response format: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from model")
	}

	text := geminiResp.Candidates[0].Content.Parts[0].Text
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	
	return text, nil
}