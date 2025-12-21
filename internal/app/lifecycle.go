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
	data, _ := game.FetchUnits(client)

	config.UpdateConfig(func(cfg *config.Config) {
		cfg.BlurredBackground = graphics.BlurScreenshot()
	})

	stateData := loadStateData(client)

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

	_ = ebiten.RunGame(gameInstance)

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

func loadStateData(client *pepeunit.PepeunitClient) map[string][][]string {
	stateData := make(map[string][][]string)
	if client == nil {
		return stateData
	}

	ctx := context.Background()
	stateStr, err := client.GetStateStorage(ctx)
	if err != nil || stateStr == "" || stateStr == "\"\"" {
		return stateData
	}

	if err := json.Unmarshal([]byte(stateStr), &stateData); err == nil {
		return stateData
	}

	var wrapped string
	if err := json.Unmarshal([]byte(stateStr), &wrapped); err != nil || wrapped == "" {
		return stateData
	}
	_ = json.Unmarshal([]byte(wrapped), &stateData)
	return stateData
}
