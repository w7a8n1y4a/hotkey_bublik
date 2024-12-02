package schema

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// Schema структура для представления JSON schema
type Schema struct {
	InputBaseTopic  map[string][]string `json:"input_base_topic"`
	OutputBaseTopic map[string][]string `json:"output_base_topic"`
	InputTopic      map[string][]string `json:"input_topic"`
	OutputTopic     map[string][]string `json:"output_topic"`
}

// LoadSchema читает JSON schema из файла
func LoadSchema() (*Schema, error) {
	file, err := os.Open("schema.json")
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл: %w", err)
	}
	defer file.Close()

	var schema Schema
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла: %w", err)
	}

	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	return &schema, nil
}

// SaveSchema записывает обновленную JSON schema в файл
func SaveSchema(schema Schema) error {
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %w", err)
	}

	if err := os.WriteFile("schema.json", data, 0644); err != nil {
		return fmt.Errorf("ошибка записи в файл: %w", err)
	}

    fmt.Println("Новая схема успешно записана")

	return nil
}
