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

type AzureOpenAI struct {
	config            gptl.Config
	historyRepository gptl.HisotryRepository
}

func (p AzureOpenAI) Chat(input io.Reader, output io.Writer, option ...gptl.ChatOption) error {
	options := gptl.NewChatOptions(option...)

	messages := []OpenAIMessage{}
	err := p.historyRepository.LoadHistory(p.config.Provider, options.History, &messages)
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
		Model:    p.config.Model,
		Messages: messages,
	}

	chatCompletationPath, err := url.JoinPath(p.config.Endpoint, "openai", "deployments", p.config.Model, "chat", "completions")
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
	postReq.Header.Add("api-key", p.config.ApiKey)
	q := postReq.URL.Query()
	q.Set("api-version", "2023-05-15")
	postReq.URL.RawQuery = q.Encode()

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

	if err := p.historyRepository.SaveHistory(p.config.Provider, options.History, messages); err != nil {
		return err
	}

	return nil
}

func NewAzureOpenAI(config gptl.Config, historyRepository gptl.HisotryRepository) (gptl.Provider, error) {
	if config.Model == "" {
		config.Model = "gpt-4o-mini"
	}

	return &AzureOpenAI{
		config:            config,
		historyRepository: historyRepository,
	}, nil
}

func init() {
	gptl.RegisterProviderFactory("azure-openai", NewAzureOpenAI)
}
