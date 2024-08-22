package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	config            Config
	hisotryRepository HisotryRepository
}

func (p OpenAIProvider) Chat(input io.Reader, output io.Writer, option ...ChatOption) error {
	options := NewChatOptions(option...)

	messages := []OpenAIMessage{}
	if err := p.hisotryRepository.LoadHistory(p.config.Provider, options.History, &messages); err != nil {
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
		Model:    p.config.Model,
		Messages: messages,
	}

	chatCompletationPath, err := url.JoinPath(p.config.Endpoint, "chat", "completions")
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
	postReq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.config.ApiKey))

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

	if err := p.hisotryRepository.SaveHistory(p.config.Provider, options.History, messages); err != nil {
		return err
	}

	return nil
}

func NewOpenAIProvider(config Config, historyRepository HisotryRepository) (Provider, error) {
	if config.Endpoint == "" {
		config.Endpoint = "https://api.openai.com/v1"
	}

	if config.Model == "" {
		config.Model = "gpt-3.5-turbo"
	}

	return &OpenAIProvider{
		config:            config,
		hisotryRepository: historyRepository,
	}, nil
}

func init() {
	RegisterProviderFactory("openai", NewOpenAIProvider)
}
