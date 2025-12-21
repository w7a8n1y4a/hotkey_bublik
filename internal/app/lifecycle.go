package app

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"

	"picker/internal/config"
	"picker/internal/game"
	"picker/internal/graphics"

	"github.com/hajimehoshi/ebiten/v2"
	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

func prepareGame(client *pepeunit.PepeunitClient) (*game.Game, error) {
	data, err := game.FetchUnits(client)
	if err != nil {
		// Ошибка при получении данных (используем пустой список юнитов)
	}

	config.UpdateConfig(func(cfg *config.Config) {
		cfg.BlurredBackground = graphics.BlurScreenshot()
	})

	stateData := make(map[string][][]string)
	ctx := context.Background()
	if stateStr, err := client.GetStateStorage(ctx); err == nil && stateStr != "" && stateStr != "\"\"" {
		if err := json.Unmarshal([]byte(stateStr), &stateData); err != nil {
			var wrappedStr string
			if err2 := json.Unmarshal([]byte(stateStr), &wrappedStr); err2 == nil && wrappedStr != "" {
				if err3 := json.Unmarshal([]byte(wrappedStr), &stateData); err3 != nil {
				}
			}
		}
	}

	for range stateData {
	}

	gameInstance, err := game.NewGame(client, data, stateData)
	if err != nil {
		return nil, err
	}

	gameInstance.OnHotkeysChanged = func() {
		go func() {
			RegisterOptionHotkeys(client)
		}()
	}

	return gameInstance, nil
}

func StartGame(client *pepeunit.PepeunitClient) {
	gameInstance, err := prepareGame(client)
	if err != nil {
		return
	}

	ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("Picker")

	if err := ebiten.RunGame(gameInstance); err != nil {
		// Ошибка запуска игры
	}

	restartApplication()
}

func restartApplication() {
	exe, err := os.Executable()
	if err != nil {
		os.Exit(1)
	}
	cmd := exec.Command(exe)
	if err := cmd.Start(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
