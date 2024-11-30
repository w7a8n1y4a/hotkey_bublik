package queries

import (
	"fmt"
    "strings"
    "encoding/json"
	"io/ioutil"
	"net/http"
     
	"picker/internal/config"
)

// Определение структур для десериализации JSON-ответа

type UnitNode struct {
	UUID             string `json:"uuid"`
	Type             string `json:"type"`
	VisibilityLevel  string `json:"visibility_level"`
	IsRewritableInput bool   `json:"is_rewritable_input"`
	TopicName        string `json:"topic_name"`
	CreateDatetime   string `json:"create_datetime"`
	State            string `json:"state"`
	UnitUUID         string `json:"unit_uuid"`
	CreatorUUID      string `json:"creator_uuid"`
}

type Unit struct {
	UUID                      string     `json:"uuid"`
	VisibilityLevel           string     `json:"visibility_level"`
	Name                      string     `json:"name"`
	CreateDatetime            string     `json:"create_datetime"`
	IsAutoUpdateFromRepoUnit  bool       `json:"is_auto_update_from_repo_unit"`
	RepoBranch                string     `json:"repo_branch"`
	RepoCommit                string     `json:"repo_commit"`
	UnitStateDict             string     `json:"unit_state_dict"`
	CurrentCommitVersion      string     `json:"current_commit_version"`
	LastUpdateDatetime        string     `json:"last_update_datetime"`
	CreatorUUID               string     `json:"creator_uuid"`
	RepoUUID                  string     `json:"repo_uuid"`
	UnitNodes                 []UnitNode `json:"unit_nodes"`
}

type UnitsByNodesResponse struct {
	Count int    `json:"count"`
	Units []Unit `json:"units"`
}

func GetUnitsByNodesQuery() (unitsByNodes UnitsByNodesResponse, err error){

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
    
    unitNodeUUIDs := []string{
        "b5bb0caa-e01f-4940-97da-a8400c1c5ed6",
        "4a7d0592-05cf-4360-a6bc-b6c95f5e146b",
    }

	// Добавляем массив unitNodeUUIDs в параметры
	for _, uuid := range unitNodeUUIDs {
		params = append(params, "unit_node_uuids="+uuid)
	}

	// Собираем полный URL
	fullURL := baseURL + "?" + strings.Join(params, "&")
    fmt.Println(fullURL) 

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
    
    return
}


