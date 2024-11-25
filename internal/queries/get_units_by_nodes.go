package queries

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

type ApiResponse struct {
	Count int    `json:"count"`
	Units []Unit `json:"units"`
}

func TestQuery() {
	// URL запроса
    url := "https://devunit.pepeunit.com/pepeunit/api/v1/units?is_include_output_unit_nodes=true&unit_node_uuids=b5bb0caa-e01f-4940-97da-a8400c1c5ed6&unit_node_uuids=4a7d0592-05cf-4360-a6bc-b6c95f5e146b&visibility_level=Public&visibility_level=Internal&visibility_level=Private&order_by_unit_name=asc&order_by_create_date=desc&order_by_last_update=desc&unit_node_type=Output&unit_node_type=Input"
	// Создание HTTP-запроса
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Ошибка создания запроса:", err)
		return
	}

	// Установка заголовков
	req.Header.Set("accept", "application/json")

	// Отправка запроса
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Ошибка выполнения запроса:", err)
		return
	}
	defer resp.Body.Close()

	// Чтение ответа
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Ошибка чтения ответа:", err)
		return
	}

	// Десериализация JSON
	var apiResponse ApiResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		fmt.Println("Ошибка декодирования JSON:", err)
		return
	}

	// Вывод результата
	fmt.Printf("Общее количество: %d\n", apiResponse.Count)
	for _, unit := range apiResponse.Units {
		fmt.Printf("UUID: %s, Name: %s, Create Date: %s\n", unit.UUID, unit.Name, unit.CreateDatetime)
		for _, node := range unit.UnitNodes {
			fmt.Printf("\tNode UUID: %s, Type: %s, Topic: %s\n", node.UUID, node.Type, node.TopicName)
		}
	}
}


