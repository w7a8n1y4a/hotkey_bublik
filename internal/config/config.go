package config

import (
	"encoding/json"
	"image"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

// Config хранит только внутренние настройки приложения (геометрия и графика).
// Все сетевые и платформенные настройки берутся из pepeunit_go_client.
type Config struct {
	// Внутренние переменные приложения
	ScreenWidth       int          `json:"-"`
	ScreenHeight      int          `json:"-"`
	PickerCenterX     int          `json:"-"`
	PickerCenterY     int          `json:"-"`
	RadiusInner       int          `json:"RADIUS_INNER"`
	ThickSegment      int          `json:"THICK_SEGMENT"`
	BlurredBackground *image.NRGBA `json:"-"`
}

// Глобальная переменная для конфигурации
var config Config

// вспомогательная структура только для чтения значений бублика из env.json
type donutEnv struct {
	RadiusInner  int `json:"RADIUS_INNER"`
	ThickSegment int `json:"THICK_SEGMENT"`
}

// Инициализация пакета
func init() {
	// Геометрия экрана
	config.ScreenWidth, config.ScreenHeight = ebiten.Monitor().Size()
	config.PickerCenterX, config.PickerCenterY = config.ScreenWidth/2, config.ScreenHeight/2
	// Значения по умолчанию для радиуса бублика, если в env.json их нет
	config.RadiusInner = 200
	config.ThickSegment = 50

	// Пытаемся загрузить RADIUS_INNER и THICK_SEGMENT из общего env.json,
	// остальные переменные читает pepeunit_go_client.
	_ = loadDonutConfigFromFile("env.json")
}

// loadDonutConfigFromFile загружает только параметры бублика из env.json
func loadDonutConfigFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		// env.json может отсутствовать до первой синхронизации – это не критично
		return err
	}
	defer file.Close()

	var env donutEnv
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&env); err != nil {
		return err
	}

	if env.RadiusInner > 0 {
		config.RadiusInner = env.RadiusInner
	}
	if env.ThickSegment > 0 {
		config.ThickSegment = env.ThickSegment
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
