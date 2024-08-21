package main

import (
	"os"
	"path/filepath"
)

func SaveHistory(provider string, historyFileName string, content []byte) error {
	dir := filepath.Join(historyDirPath, provider)

	err := os.MkdirAll(dir, 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}

	if err := os.WriteFile(filepath.Join(dir, historyFileName), content, 0600); err != nil {
		return err
	}

	return nil
}

func LoadHistory(provider string, historyFileName string) ([]byte, error) {
	dir := filepath.Join(historyDirPath, provider)

	err := os.MkdirAll(dir, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}

	content, err := os.ReadFile(filepath.Join(dir, historyFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return []byte{}, nil
		}

		return nil, err
	}

	return content, nil
}
