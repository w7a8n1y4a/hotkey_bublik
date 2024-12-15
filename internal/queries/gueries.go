package queries

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"picker/internal/config"
	"picker/internal/schema"
	"strings"
    "archive/zip"
    "io"
    "os"
    "path/filepath"
)

// Определение структур для десериализации JSON-ответа

type UnitNode struct {
	UUID              string `json:"uuid"`
	Type              string `json:"type"`
	VisibilityLevel   string `json:"visibility_level"`
	IsRewritableInput bool   `json:"is_rewritable_input"`
	TopicName         string `json:"topic_name"`
	CreateDatetime    string `json:"create_datetime"`
	State             string `json:"state"`
	UnitUUID          string `json:"unit_uuid"`
	CreatorUUID       string `json:"creator_uuid"`
}

type Unit struct {
	UUID                     string     `json:"uuid"`
	VisibilityLevel          string     `json:"visibility_level"`
	Name                     string     `json:"name"`
	CreateDatetime           string     `json:"create_datetime"`
	IsAutoUpdateFromRepoUnit bool       `json:"is_auto_update_from_repo_unit"`
	RepoBranch               string     `json:"repo_branch"`
	RepoCommit               string     `json:"repo_commit"`
	UnitStateDict            string     `json:"unit_state_dict"`
	CurrentCommitVersion     string     `json:"current_commit_version"`
	LastUpdateDatetime       string     `json:"last_update_datetime"`
	CreatorUUID              string     `json:"creator_uuid"`
	RepoUUID                 string     `json:"repo_uuid"`
	UnitNodes                []UnitNode `json:"unit_nodes"`
}

type UnitsByNodesResponse struct {
	Count int    `json:"count"`
	Units []Unit `json:"units"`
}

type UnitNodesResponse struct {
   	Count int            `json:"count"`
	UnitNodes []UnitNode `json:"unit_nodes"` 
}

func extractUUIDs(urls []string) []string {
	uuids := []string{}
	for _, url := range urls {
		parts := strings.Split(url, "/")
		if len(parts) > 1 {
			uuids = append(uuids, parts[1])
		}
	}
	return uuids
}

func GetInputByOutput() (unitNodes UnitNodesResponse, err error) {

	cfg := config.GetConfig()

	// Формируем параметры запроса
	baseURL := fmt.Sprintf("%s://%s/pepeunit/api/v1/unit_nodes", cfg.HTTP_TYPE, cfg.PEPEUNIT_URL)
    
    schemaData, err := schema.LoadSchema()
    
    uuid := strings.Split(schemaData.OutputTopic["output_units_nodes/pepeunit"][0], "/")[1]

	// Добавляем параметры в URL
	params := []string{
		"visibility_level=Public",
		"visibility_level=Internal",
		"visibility_level=Private",
		"order_by_create_date=desc",
		"type=Output",
		"type=Input",
        "output_uuid=" + uuid,
	}

	// Собираем полный URL
	fullURL := baseURL + "?" + strings.Join(params, "&")

	// f Создание HTTP-запроса
	req, err := http.NewRequest("GET", fullURL, nil)

	// Установка заголовков
	req.Header.Set("accept", "application/json")
	req.Header.Set("x-auth-token", cfg.PEPEUNIT_TOKEN)

	// Отправка запроса
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Чтение ответа
	body, err := ioutil.ReadAll(resp.Body)

	// Десериализация JSON
	err = json.Unmarshal(body, &unitNodes)

	return
}


func GetUnitsByNodesQuery() (unitsByNodes UnitsByNodesResponse, err error) {

	cfg := config.GetConfig()

	// Формируем параметры запроса
	baseURL := fmt.Sprintf("%s://%s/pepeunit/api/v1/units", cfg.HTTP_TYPE, cfg.PEPEUNIT_URL)

	// Добавляем параметры в URL
	params := []string{
		"is_include_output_unit_nodes=true",
		"visibility_level=Public",
		"visibility_level=Internal",
		"visibility_level=Private",
		"order_by_unit_name=asc",
		"order_by_create_date=desc",
		"order_by_last_update=desc",
		"unit_node_type=Output",
		"unit_node_type=Input",
	}

    unitNodes, err := GetInputByOutput()

	// Добавляем массив unitNodeUUIDs в параметры
	for _, item := range unitNodes.UnitNodes {
		params = append(params, "unit_node_uuids=" + item.UUID)
	}

	// Собираем полный URL
	fullURL := baseURL + "?" + strings.Join(params, "&")

	// f Создание HTTP-запроса
	req, err := http.NewRequest("GET", fullURL, nil)

	// Установка заголовков
	req.Header.Set("accept", "application/json")
	req.Header.Set("x-auth-token", cfg.PEPEUNIT_TOKEN)

	// Отправка запроса
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Чтение ответа
	body, err := ioutil.ReadAll(resp.Body)

	// Десериализация JSON
	err = json.Unmarshal(body, &unitsByNodes)
    for inc, item := range unitsByNodes.Units{
    
        fmt.Println(inc, item.Name)
        for i, two := range item.UnitNodes{
        
            fmt.Println(i, two.TopicName)
        }
    }
	return
}

func GetCurrentSchema() (newSchema schema.Schema, err error) {

	cfg := config.GetConfig()

	// Формируем параметры запроса
	baseURL := fmt.Sprintf("%s://%s/pepeunit/api/v1/units/get_current_schema/", cfg.HTTP_TYPE, cfg.PEPEUNIT_URL)

	// Собираем полный URL
	fullURL := baseURL + cfg.UnitUUID

	// f Создание HTTP-запроса
	req, err := http.NewRequest("GET", fullURL, nil)

	// Установка заголовков
	req.Header.Set("accept", "application/json")
	req.Header.Set("x-auth-token", cfg.PEPEUNIT_TOKEN)

	// Отправка запроса
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Чтение ответа
	body, err := ioutil.ReadAll(resp.Body)

	rawJSON := string(body)
	rawJSON = strings.Replace(rawJSON, "\\\"", "\"", -1)

	// Десериализация JSON
	err = json.Unmarshal([]byte(rawJSON[1:len(rawJSON)-1]), &newSchema)

	return
}

func GetCurrentVersion() (path string, err error) {
	cfg := config.GetConfig()

	// Формируем параметры запроса
	baseURL := fmt.Sprintf("%s://%s/pepeunit/api/v1/units/firmware/zip/%s", cfg.HTTP_TYPE, cfg.PEPEUNIT_URL, cfg.UnitUUID)

	// Создание HTTP-запроса
	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Установка заголовков
	req.Header.Set("accept", "application/json")
	req.Header.Set("x-auth-token", cfg.PEPEUNIT_TOKEN)

	// Отправка запроса
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}

	// Создаем временный файл для сохранения архива
	tempFile, err := os.CreateTemp("", "firmware_*.zip")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name()) // Удаляем временный файл после использования

	// Сохраняем содержимое ответа в файл
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save response to file: %w", err)
	}

	// Закрываем файл для последующего чтения
	tempFile.Close()

	// Директория для распаковки
	outputDir := "update_data"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Распаковка архива
	err = unzip(tempFile.Name(), outputDir)
	if err != nil {
		return "", fmt.Errorf("failed to unzip file: %w", err)
	}

	return outputDir, nil
}

func unzip(src string, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		filePath := filepath.Join(dest, file.Name)

		// Проверяем, что путь не выходит за пределы целевой директории
		if !filepath.HasPrefix(filePath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", filePath)
		}

		if file.FileInfo().IsDir() {
			// Создаем директории
			if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		} else {
			// Распаковываем файлы
			if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
				return fmt.Errorf("failed to create directory for file: %w", err)
			}

			outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
			if err != nil {
				return fmt.Errorf("failed to open file for writing: %w", err)
			}

			fileReader, err := file.Open()
			if err != nil {
				return fmt.Errorf("failed to open file inside zip: %w", err)
			}

			_, err = io.Copy(outFile, fileReader)
			outFile.Close()
			fileReader.Close()
			if err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
		}
	}

	return nil
}

