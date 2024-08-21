package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type ClaudeProvider struct {
	config Config
}

func (c ClaudeProvider) loadMessages(options ChatOptions) ([]map[string]interface{}, error) {
	var content []byte
	var err error

	if options.History != "" {
		content, err = LoadHistory(c.GetName(), options.History+".yaml")
	} else if options.Template != "" {
		content, err = os.ReadFile(filepath.Join(templateDirPath, c.GetName(), fmt.Sprintf("%s.yaml", options.Template)))
	} else {
		return []map[string]interface{}{}, nil
	}

	if err != nil {
		return nil, err
	}

	var messages []map[string]interface{}
	if err := yaml.Unmarshal(content, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

// Chat implements Provider.
func (c *ClaudeProvider) Chat(input io.Reader, output io.Writer, option ...ChatOption) error {
	options := NewChatOptions(option...)
	messages, err := c.loadMessages(options)
	if err != nil {
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

	chatCompletationPath, err := url.JoinPath(c.GetEndpoint(), "messages")
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

	historyContent, err := yaml.Marshal(messages)
	if err != nil {
		return err
	}

	if err := SaveHistory(c.GetName(), options.History+".yaml", historyContent); err != nil {
		return err
	}

	return nil
}

// GetEndpoint implements Provider.
func (c *ClaudeProvider) GetEndpoint() string {
	return "https://api.anthropic.com/v1"
}

// GetModels implements Provider.
func (c *ClaudeProvider) GetModels() []string {
	return []string{
		"claude-3-5-sonnet-20240620",
	}
}

// GetName implements Provider.
func (c *ClaudeProvider) GetName() string {
	return "claude"
}

// SetConfig implements Provider.
func (c *ClaudeProvider) SetConfig(config Config) {
	c.config = config

	if c.config.MaxTokens <= 0 {
		c.config.MaxTokens = 1024
	}
}
