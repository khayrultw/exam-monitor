package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type FormData struct {
	StudentID string `json:"student_id"`
	Name      string `json:"name"`
	Room      string `json:"room"`
}

func getDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dataDir := filepath.Join(home, ".exam-monitor")
	err = os.MkdirAll(dataDir, 0755)
	return dataDir, err
}

func SaveFormData(studentID, name, room string) error {
	dataDir, err := getDataDir()
	if err != nil {
		return err
	}

	data := FormData{
		StudentID: studentID,
		Name:      name,
		Room:      room,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(dataDir, "form_data.json")
	return os.WriteFile(filePath, jsonData, 0644)
}

func LoadFormData() (*FormData, error) {
	dataDir, err := getDataDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(dataDir, "form_data.json")
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &FormData{}, nil
		}
		return nil, err
	}

	var data FormData
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}
