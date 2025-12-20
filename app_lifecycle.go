package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"log"
	"os"
	"os/exec"

	"picker/internal/config"
	"picker/internal/game"
	"picker/internal/graphics"

	"github.com/hajimehoshi/ebiten/v2"
	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

func loadIcon(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var icon []byte
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		return nil, err
	}
	icon = buf.Bytes()
	return icon, nil
}

func prepareGame(client *pepeunit.PepeunitClient) (*game.Game, error) {
	log.Println("DEBUG: prepareGame called")
	data, err := game.FetchUnits(client)
	if err != nil {
		log.Printf("Ошибка при получении данных (используем пустой список юнитов): %v", err)
	}
	log.Printf("DEBUG: loaded %d units\n", len(data.Units))

	config.UpdateConfig(func(cfg *config.Config) {
		cfg.BlurredBackground = graphics.BlurScreenshot()
	})

	stateData := make(map[string][][]string)
	ctx := context.Background()
	if stateStr, err := client.GetStateStorage(ctx); err == nil && stateStr != "" && stateStr != "\"\"" {
		log.Printf("DEBUG: loading state data: %s\n", stateStr)

		// Попробуем распарсить как прямой JSON объект
		if err := json.Unmarshal([]byte(stateStr), &stateData); err != nil {
			log.Printf("DEBUG: failed to unmarshal state as object: %v\n", err)

			// Попробуем распарсить как JSON-строку внутри JSON-строки
			var wrappedStr string
			if err2 := json.Unmarshal([]byte(stateStr), &wrappedStr); err2 == nil && wrappedStr != "" {
				log.Printf("DEBUG: unwrapped state string: %s\n", wrappedStr)
				if err3 := json.Unmarshal([]byte(wrappedStr), &stateData); err3 != nil {
					log.Printf("DEBUG: failed to unmarshal unwrapped state: %v\n", err3)
				} else {
					log.Printf("DEBUG: successfully loaded state from wrapped string\n")
				}
			} else {
				log.Printf("DEBUG: failed to unmarshal as string wrapper: %v\n", err2)
			}
		} else {
			log.Printf("DEBUG: successfully loaded state as direct JSON object\n")
		}
	} else {
		log.Printf("DEBUG: no state data loaded: err=%v, stateStr='%s'\n", err, stateStr)
	}

	log.Printf("DEBUG: loaded state data for %d nodes\n", len(stateData))
	for nodeUUID, options := range stateData {
		log.Printf("DEBUG: node %s has %d options\n", nodeUUID, len(options))
	}

	gameInstance, err := game.NewGame(client, data, stateData)
	if err != nil {
		return nil, err
	}

	log.Printf("DEBUG: Game created with %d units, %d state nodes\n", len(data.Units), len(stateData))
	return gameInstance, nil
}

func startGame(client *pepeunit.PepeunitClient) {
	gameInstance, err := prepareGame(client)
	if err != nil {
		log.Fatalf("Ошибка при подготовке игры: %v", err)
		return
	}

	ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("Picker")

	if err := ebiten.RunGame(gameInstance); err != nil {
		log.Printf("Ошибка запуска игры: %v", err)
	}

	restartApplication()
}

func restartApplication() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatal("Не удалось получить исполнимый файл:", err)
	}
	cmd := exec.Command(exe)
	if err := cmd.Start(); err != nil {
		log.Fatal("Не удалось перезапустить приложение:", err)
	}
	os.Exit(0)
}
