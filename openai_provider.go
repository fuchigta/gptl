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

type OpenAIMessage struct {
	Role    string `json:"role" yaml:"role"`
	Content string `json:"content" yaml:"content"`
}

type OpenAIReq struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
}

type OpenAIResChoice struct {
	Message OpenAIMessage `json:"message"`
}

type OpenAIResError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

type OpenAIRes struct {
	Choices []OpenAIResChoice `json:"choices"`
	Error   OpenAIResError    `json:"error"`
}

type OpenAIProvider struct {
	Name     string
	Models   []string
	Endpoint string
	Config   Config
}

func (p OpenAIProvider) GetName() string {
	return p.Name
}

func (p OpenAIProvider) GetModels() []string {
	return p.Models
}

func (p OpenAIProvider) GetEndpoint() string {
	return p.Endpoint
}

func (p *OpenAIProvider) SetConfig(config Config) {
	p.Config = config
}

func (p OpenAIProvider) loadMessages(options ChatOptions) ([]OpenAIMessage, error) {
	var content []byte
	var err error

	if options.History != "" {
		content, err = LoadHistory(p.GetName(), options.History+".yaml")
	} else if options.Template != "" {
		content, err = os.ReadFile(filepath.Join(templateDirPath, p.GetName(), fmt.Sprintf("%s.yaml", options.Template)))
	} else {
		return []OpenAIMessage{}, nil
	}

	if err != nil {
		return nil, err
	}

	var messages []OpenAIMessage
	if err := yaml.Unmarshal(content, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func (p OpenAIProvider) Chat(input io.Reader, output io.Writer, option ...ChatOption) error {
	options := NewChatOptions(option...)
	messages, err := p.loadMessages(options)
	if err != nil {
		return err
	}

	userMsgBytes, err := io.ReadAll(input)
	if err != nil {
		return err
	}

	var userMsg OpenAIMessage
	if err := json.Unmarshal(userMsgBytes, &userMsg); err != nil {
		userMsg = OpenAIMessage{
			Role:    "user",
			Content: string(userMsgBytes),
		}
	}

	messages = append(messages, userMsg)

	client := http.Client{}

	req := OpenAIReq{
		Model:    p.Config.Model,
		Messages: messages,
	}

	chatCompletationPath, err := url.JoinPath(p.GetEndpoint(), "chat", "completions")
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
	postReq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.Config.ApiKey))

	res, err := client.Do(postReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var resContent OpenAIRes
	err = json.NewDecoder(res.Body).Decode(&resContent)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s (%s/%s)", resContent.Error.Message, resContent.Error.Code, resContent.Error.Type)
	}

	for _, choice := range resContent.Choices {
		fmt.Fprintln(output, choice.Message.Content)
		messages = append(messages, choice.Message)
	}

	historyContent, err := yaml.Marshal(messages)
	if err != nil {
		return err
	}

	if err := SaveHistory(p.GetName(), options.History+".yaml", historyContent); err != nil {
		return err
	}

	return nil
}
