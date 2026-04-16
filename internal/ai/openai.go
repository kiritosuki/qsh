package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Response struct {
	Command     string `json:"command"`
	Explanation string `json:"explanation"`
	Risk        string `json:"risk"`
}

func Query(apiKey, userInput string) (*Response, error) {

	prompt := fmt.Sprintf(`
You are a Linux shell expert.

User input: "%s"

Return JSON only:
{
  "command": "...",
  "explanation": "...",
  "risk": "low|medium|high"
}

Rules:
- If user asks for explanation, command can be empty
- If generating command, keep it safe
- No extra text
`, userInput)

	reqBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	content := result["choices"].([]interface{})[0].(map[string]interface{})["message"].(map[string]interface{})["content"].(string)

	var parsed Response
	err = json.Unmarshal([]byte(content), &parsed)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %v", err)
	}

	return &parsed, nil
}
