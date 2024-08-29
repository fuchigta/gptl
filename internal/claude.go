package gptl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/fuchigta/gptl"
)

type Claude struct {
	config            gptl.Config
	historyRepository gptl.HisotryRepository
}

// Chat implements Provider.
func (c *Claude) Chat(input io.Reader, output io.Writer, option ...gptl.ChatOption) error {
	options := gptl.NewChatOptions(option...)

	messages := []map[string]interface{}{}
	if err := c.historyRepository.LoadHistory(c.config.Provider, options.History, &messages); err != nil {
		return err
	}

	userMsgBytes, err := io.ReadAll(input)
	if err != nil {
		return err
	}

	var userMsg map[string]interface{}
	if err := json.Unmarshal(userMsgBytes, &userMsg); err != nil {
		userMsg = map[string]interface{}{
			"role":    "user",
			"content": string(userMsgBytes),
		}
	}

	messages = append(messages, userMsg)
	client := http.Client{}

	req := map[string]interface{}{
		"model":      c.config.Model,
		"messages":   messages,
		"max_tokens": c.config.MaxTokens,
	}

	chatCompletationPath, err := url.JoinPath(c.config.Endpoint, "messages")
	if err != nil {
		return err
	}

	bodyContent, err := json.Marshal(&req)
	if err != nil {
		return err
	}

	postReq, err := http.NewRequest(http.MethodPost, chatCompletationPath, bytes.NewBuffer(bodyContent))
	if err != nil {
		return err
	}

	postReq.Header.Add("Content-Type", "application/json")
	postReq.Header.Add("x-api-key", c.config.ApiKey)
	postReq.Header.Add("anthropic-version", "2023-06-01")

	res, err := client.Do(postReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var resContent struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
		Role string `json:"role"`
	}
	err = json.NewDecoder(res.Body).Decode(&resContent)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(resContent.Error.Message)
	}

	for _, content := range resContent.Content {
		fmt.Fprintln(output, content.Text)
		messages = append(messages, map[string]interface{}{
			"role":    resContent.Role,
			"content": content.Text,
		})
	}

	if err := c.historyRepository.SaveHistory(c.config.Provider, options.History, messages); err != nil {
		return err
	}

	return nil
}

func NewClaude(config gptl.Config, historyRepository gptl.HisotryRepository) (gptl.Provider, error) {
	if config.Endpoint == "" {
		config.Endpoint = "https://api.anthropic.com/v1"
	}

	if config.Model == "" {
		config.Model = "claude-3-5-sonnet-20240620"
	}

	if config.MaxTokens <= 0 {
		config.MaxTokens = 1024
	}

	return &Claude{
		config:            config,
		historyRepository: historyRepository,
	}, nil
}

func init() {
	gptl.RegisterProviderFactory("claude", NewClaude)
}
