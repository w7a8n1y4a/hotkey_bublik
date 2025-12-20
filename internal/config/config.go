package config

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type Config struct {
	ScreenWidth       int           `json:"-"`
	ScreenHeight      int           `json:"-"`
	PickerCenterX     int           `json:"-"`
	PickerCenterY     int           `json:"-"`
	RadiusInner       int           `json:"RADIUS_INNER"`
	ThickSegment      int           `json:"THICK_SEGMENT"`
	BlurredBackground *ebiten.Image `json:"-"`
	LaunchHotkeyMain  *string       `json:"HOTKEY_MAIN"`
}

var config Config

var envFilePath = "env.json"

var lastEnvModTime time.Time

type donutEnv struct {
	RadiusInner  int     `json:"RADIUS_INNER"`
	ThickSegment int     `json:"THICK_SEGMENT"`
	HotkeyMain   *string `json:"HOTKEY_MAIN"`
}

func init() {
	config.ScreenWidth, config.ScreenHeight = ebiten.Monitor().Size()
	config.PickerCenterX, config.PickerCenterY = config.ScreenWidth/2, config.ScreenHeight/2
	config.RadiusInner = 200
	config.ThickSegment = 50
	config.LaunchHotkeyMain = nil

	_ = loadDonutConfigFromFile(envFilePath)
}

func loadDonutConfigFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
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

	if env.HotkeyMain == nil {
		config.LaunchHotkeyMain = nil
	} else {
		trimmed := strings.TrimSpace(*env.HotkeyMain)
		if trimmed == "" {
			config.LaunchHotkeyMain = nil
		} else {
			s := trimmed
			config.LaunchHotkeyMain = &s
		}
	}

	if info, statErr := os.Stat(filePath); statErr == nil {
		lastEnvModTime = info.ModTime()
	}
	return nil
}

func GetConfig() Config {
	if info, err := os.Stat(envFilePath); err == nil {
		if lastEnvModTime.IsZero() || info.ModTime().After(lastEnvModTime) {
			_ = loadDonutConfigFromFile(envFilePath)
		}
	}

	return config
}

func UpdateConfig(updateFunc func(cfg *Config)) {
	updateFunc(&config)
}
