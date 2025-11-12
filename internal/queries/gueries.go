package queries

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"picker/internal/config"
	"strings"
)

// Определение структур для десериализации JSON-ответа

type schemaFile struct {
	InputBaseTopic  map[string][]string `json:"input_base_topic"`
	OutputBaseTopic map[string][]string `json:"output_base_topic"`
	InputTopic      map[string][]string `json:"input_topic"`
	OutputTopic     map[string][]string `json:"output_topic"`
}

func loadSchema() (*schemaFile, error) {
	b, err := os.ReadFile("schema.json")
	if err != nil {
		return nil, err
	}
	var s schemaFile
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

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
	Count     int        `json:"count"`
	UnitNodes []UnitNode `json:"unit_nodes"`
}

type StateRequest struct {
	State string `json:"state"`
}

func GetInputByOutput() (unitNodes UnitNodesResponse, err error) {

	cfg := config.GetConfig()

	// Формируем параметры запроса
	baseURL := fmt.Sprintf("%s://%s%s%s/unit_nodes", cfg.HTTP_TYPE, cfg.PEPEUNIT_URL, cfg.PEPEUNIT_APP_PREFIX, cfg.PEPEUNIT_API_ACTUAL_PREFIX)

	schemaData, err := loadSchema()

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
	baseURL := fmt.Sprintf("%s://%s%s%s/units", cfg.HTTP_TYPE, cfg.PEPEUNIT_URL, cfg.PEPEUNIT_APP_PREFIX, cfg.PEPEUNIT_API_ACTUAL_PREFIX)

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

	if unitNodes.Count == 0 {
		return UnitsByNodesResponse{}, fmt.Errorf("No edges found")
	}

	// Добавляем массив unitNodeUUIDs в параметры
	for _, item := range unitNodes.UnitNodes {
		params = append(params, "unit_node_uuids="+item.UUID)
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
	for inc, item := range unitsByNodes.Units {

		fmt.Println(inc, item.Name)
		for i, two := range item.UnitNodes {

			fmt.Println(i, two.TopicName)
		}
	}
	return
}
