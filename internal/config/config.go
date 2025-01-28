package config

import (
	"encoding/json"
	"errors"
	"os"
    "encoding/base64"
	"strings"
    "image"

	"github.com/hajimehoshi/ebiten/v2"
)

// UnifiedConfig объединяет переменные конфигурации и приложения
type Config struct {
	// Переменные из env.json
	PEPEUNIT_URL        string `json:"PEPEUNIT_URL"`
	PEPEUNIT_APP_PREFIX string `json:"PEPEUNIT_APP_PREFIX"`
	PEPEUNIT_API_ACTUAL_PREFIX string `json:"PEPEUNIT_API_ACTUAL_PREFIX"`
	HTTP_TYPE           string `json:"HTTP_TYPE"`
	MQTT_URL            string `json:"MQTT_URL"`
	MQTT_PORT           int    `json:"MQTT_PORT"`
	PEPEUNIT_TOKEN      string `json:"PEPEUNIT_TOKEN"`
	SYNC_ENCRYPT_KEY    string `json:"SYNC_ENCRYPT_KEY"`
	SECRET_KEY          string `json:"SECRET_KEY"`
	PING_INTERVAL       int    `json:"PING_INTERVAL"`
	STATE_SEND_INTERVAL int    `json:"STATE_SEND_INTERVAL"`
    COMMIT_VERSION      string `json:"COMMIT_VERSION"`

	// Внутренние переменные приложения
	ScreenWidth       int            `json:"-"`
	ScreenHeight      int            `json:"-"`
	PickerCenterX     int            `json:"-"`
	PickerCenterY     int            `json:"-"`
	RadiusInner       int            `json:"RADIUS_INNER"`
	ThickSegment      int            `json:"THICK_SEGMENT"`
	BlurredBackground *image.NRGBA  `json:"-"`
    UnitUUID          string         `json:"-"`
}

type Payload struct {
	UUID string `json:"uuid"`
	Type string `json:"type"`
}

// Глобальная переменная для объединённой конфигурации
var config Config

// Инициализация пакета
func init() {
    config.ScreenWidth, config.ScreenHeight = ebiten.Monitor().Size()
	config.PickerCenterX, config.PickerCenterY = config.ScreenWidth/2, config.ScreenHeight/2
 

	// Загрузка конфигурации из env.json
	if err := loadConfigFromFile("env.json"); err != nil {
		panic(err) // Если не удалось загрузить конфигурацию, программа завершится с ошибкой
	}
}

func getUuidFromToken(jwt string) (string, error) {
	// Разделяем JWT на части
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return "", errors.New("неверный формат JWT")
	}

	// Декодируем полезную нагрузку (вторая часть)
	payloadBase64 := parts[1]
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadBase64)
	if err != nil {
		return "", errors.New("ошибка декодирования Base64: " + err.Error())
	}

	// Распарсим JSON в структуру
	var payload Payload
	err = json.Unmarshal(payloadBytes, &payload)
	if err != nil {
		return "", errors.New("ошибка парсинга JSON: " + err.Error())
	}

	// Возвращаем UUID
	return payload.UUID, nil
}

// loadConfigFromFile загружает конфигурацию из файла env.json
func loadConfigFromFile(filePath string) error {
	// Проверка на существование файла
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		return errors.New("файл env.json не найден")
	}

	// Открытие файла
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Декодирование JSON
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return err
	}

    config.UnitUUID, err = getUuidFromToken(config.PEPEUNIT_TOKEN)

	// Проверка обязательных полей
	return validateConfig()
}

// validateConfig проверяет, что все обязательные поля присутствуют
func validateConfig() error {
	if config.PEPEUNIT_URL == "" ||
		config.HTTP_TYPE == "" ||
		config.MQTT_URL == "" ||
		config.PEPEUNIT_TOKEN == "" ||
		config.SYNC_ENCRYPT_KEY == "" ||
		config.SECRET_KEY == "" ||
		config.PING_INTERVAL == 0 ||
		config.STATE_SEND_INTERVAL == 0 {
		return errors.New("обязательные переменные отсутствуют или некорректны в env.json")
	}
	return nil
}

// GetConfig возвращает текущую объединённую конфигурацию
func GetConfig() Config {
	return config
}

// UpdateConfig позволяет обновлять переменные конфигурации через функцию
func UpdateConfig(updateFunc func(cfg *Config)) {
	updateFunc(&config)
}

