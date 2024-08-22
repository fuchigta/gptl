package main

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type HisotryRepository struct {
	historyDirPath string
}

func NewHistoryRepository(historyDirPath string) (HisotryRepository, error) {
	return HisotryRepository{
		historyDirPath: historyDirPath,
	}, nil
}

func (h HisotryRepository) SaveHistory(provider string, history string, messages interface{}) error {
	dir := filepath.Join(h.historyDirPath, provider)

	err := os.MkdirAll(dir, 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}

	f, err := os.Create(filepath.Join(dir, history+".yaml"))
	if err != nil {
		return err
	}
	defer f.Close()

	return yaml.NewEncoder(f).Encode(messages)
}

func (h HisotryRepository) LoadHistory(provider string, history string, messages interface{}) error {
	dir := filepath.Join(h.historyDirPath, provider)

	err := os.MkdirAll(dir, 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}

	f, err := os.Open(filepath.Join(dir, history+".yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}
	defer f.Close()

	return yaml.NewDecoder(f).Decode(messages)
}
