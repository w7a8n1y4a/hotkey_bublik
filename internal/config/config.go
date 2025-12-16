package config

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Config хранит только внутренние настройки приложения (геометрия и графика).
// Все сетевые и платформенные настройки берутся из pepeunit_go_client.
type Config struct {
	// Внутренние переменные приложения
	ScreenWidth       int           `json:"-"`
	ScreenHeight      int           `json:"-"`
	PickerCenterX     int           `json:"-"`
	PickerCenterY     int           `json:"-"`
	RadiusInner       int           `json:"RADIUS_INNER"`
	ThickSegment      int           `json:"THICK_SEGMENT"`
	BlurredBackground *ebiten.Image `json:"-"`
	LaunchHotkeyMain  *string       `json:"HOTKEY_MAIN"`
}

// Глобальная переменная для конфигурации
var config Config

// Путь до env.json, откуда берём RADIUS_INNER и THICK_SEGMENT.
var envFilePath = "env.json"

// Время последней успешной загрузки параметров бублика из env.json.
var lastEnvModTime time.Time

// вспомогательная структура только для чтения значений бублика из env.json
type donutEnv struct {
	RadiusInner  int     `json:"RADIUS_INNER"`
	ThickSegment int     `json:"THICK_SEGMENT"`
	HotkeyMain   *string `json:"HOTKEY_MAIN"`
}

// Инициализация пакета
func init() {
	// Геометрия экрана
	config.ScreenWidth, config.ScreenHeight = ebiten.Monitor().Size()
	config.PickerCenterX, config.PickerCenterY = config.ScreenWidth/2, config.ScreenHeight/2
	// Значения по умолчанию для радиуса бублика, если в env.json их нет
	config.RadiusInner = 200
	config.ThickSegment = 50
	// HOTKEY_MAIN: без дефолта из кода; пока env.json не загрузился — хоткей не регистрируем.
	config.LaunchHotkeyMain = nil

	// Пытаемся загрузить RADIUS_INNER и THICK_SEGMENT из общего env.json,
	// остальные переменные читает pepeunit_go_client.
	_ = loadDonutConfigFromFile(envFilePath)
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

	// HOTKEY_MAIN:
	// - отсутствует / null / пустая строка -> хоткей не регистрируется
	// - строка -> используем указанное значение (триммим пробелы)
	if env.HotkeyMain == nil {
		config.LaunchHotkeyMain = nil
	} else {
		trimmed := strings.TrimSpace(*env.HotkeyMain)
		if trimmed == "" {
			config.LaunchHotkeyMain = nil
		} else {
			// создаём новую строку, чтобы не держаться за память структуры env
			s := trimmed
			config.LaunchHotkeyMain = &s
		}
	}

	// Обновляем отметку времени успешной загрузки.
	if info, statErr := os.Stat(filePath); statErr == nil {
		lastEnvModTime = info.ModTime()
	}
	return nil
}

// GetConfig возвращает текущую объединённую конфигурацию
func GetConfig() Config {
	// При каждом запросе проверяем, не изменился ли env.json.
	// Если изменился — перечитываем только параметры бублика.
	if info, err := os.Stat(envFilePath); err == nil {
		// Если файл новый или его время модификации больше сохранённого — перезагружаем.
		if lastEnvModTime.IsZero() || info.ModTime().After(lastEnvModTime) {
			_ = loadDonutConfigFromFile(envFilePath)
		}
	}

	return config
}

// UpdateConfig позволяет обновлять переменные конфигурации через функцию
func UpdateConfig(updateFunc func(cfg *Config)) {
	updateFunc(&config)
}
