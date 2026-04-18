package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/kiritosuki/qsh/types"
)

const (
	LLMTimeout = 120 // 单位: s
)

type LLMClient struct {
	config   ModelConfig
	messages []Message

	StreamCallback func(string, error)

	httpClient *http.Client
}

func (c *LLMClient) Query(query string) (string, error) {
	c.messages = append(c.messages, Message{
		Role:    "user",
		Content: query,
	})

	payload := Payload{
		Model:       c.config.ModelName,
		Messages:    c.messages,
		Temperature: 0,
		Stream:      true,
	}

	message, err := c.callStream(payload)
	if err != nil {
		return "", err
	}
	c.messages = append(c.messages, message)
	return message.Content, nil
}

func (c *LLMClient) callStream(payload Payload) (Message, error) {
	req, err := c.createRequest(payload)
	if err != nil {
		return Message{}, fmt.Errorf("failed to create the request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("failed to make the API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return Message{}, fmt.Errorf("API request failed: %s", resp.Status)
	}
	content, err := c.processStream(resp)
	return Message{Role: "assistant", Content: content}, err
}

func (c *LLMClient) createRequest(payload Payload) (*http.Request, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	req, err := http.NewRequest("POST", c.config.Endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	// 判断baseURL是不是微软的格式
	if strings.Contains(c.config.Endpoint, "openai.azure.com") {
		req.Header.Set("Api-Key", c.config.Auth)
	} else {
		// 正常情况下都会走这个openAI规范分支
		// c.config.Auth即为你配置的apikey
		req.Header.Set("Authorization", "Bearer "+c.config.Auth)
	}
	if c.config.OrgID != "" {
		// 默认使用openAI的组织头 如果没配置就不设置组织头
		req.Header.Set("OpenAI-Organization", c.config.OrgID)
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *LLMClient) processStream(resp *http.Response) (string, error) {
	streamReader := bufio.NewReader(resp.Body)
	totalData := ""
	for {
		line, err := streamReader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == "data: [DONE]" {
			break
		}
		if strings.HasPrefix(line, "data:") {
			payload := strings.TrimPrefix(line, "data:")

			var responseData ResponseData
			err = json.Unmarshal([]byte(payload), &responseData)
			if err != nil {
				fmt.Println("Error parsing data:", err)
				continue
			}
			if len(responseData.Choices) == 0 {
				continue
			}
			content := responseData.Choices[0].Delta.Content
			totalData += content
			c.StreamCallback(totalData, nil)
		}
	}
	return totalData, nil
}

func NewLLMClient(config ModelConfig) *LLMClient {
	return &LLMClient{
		config: config,
		// 这里把nil转换成切片类型 创建一个不分配内存的切片 append直接在结果上分配内存
		messages: append([]Message(nil), config.Prompt...),
		httpClient: &http.Client{
			Timeout: time.Second * LLMTimeout,
		},
	}
}
