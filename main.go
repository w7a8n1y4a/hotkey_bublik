package main

import (
	"bytes"
	_ "embed"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"os/exec"
	"picker/internal/config"
	"picker/internal/game"
	"picker/internal/graphics"
	"picker/internal/queries"

	"github.com/getlantern/systray"
	"github.com/hajimehoshi/ebiten/v2"
	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
	"time"
)

//go:embed assets/icons/64.png
var iconData []byte

func main() {
	pepeClient, err := pepeunit.NewPepeunitClient(pepeunit.PepeunitClientConfig{
		EnvFilePath:       "env.json",
		SchemaFilePath:    "schema.json",
		LogFilePath:       "log.json",
		EnableMQTT:        true,
		EnableREST:        true,
		CycleSpeed:        100 * time.Millisecond,
		RestartMode:       pepeunit.RestartModeRestartExec,
	})
	if err != nil {
		log.Fatalf("init pepeunit client failed: %v", err)
	}

	ctx := context.Background()
	if pepeClient.GetMQTTClient() != nil {
		if err := pepeClient.GetMQTTClient().Connect(ctx); err != nil {
			log.Fatalf("mqtt connect failed: %v", err)
		}
	}
	go pepeClient.RunMainCycle(ctx, nil)

	// Initialize the hotkey listener in the main thread
	mainthread.Init(func() {
		registerGlobalHotkey(pepeClient)
	})

	// Prepare icon and other resources
	icon, err := loadIcon(iconData)
	if err != nil {
		log.Fatal("Ошибка загрузки иконки:", err)
	}

	// Start the system tray app
	systray.Run(func() {
		onReady(icon, pepeClient)
	}, onExit)
}

// Register the global hotkey (Ctrl + Shift + H)
func registerGlobalHotkey(client *pepeunit.PepeunitClient) {
	hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}, hotkey.KeyH)

	err := hk.Register()
	if err != nil {
		log.Fatalf("hotkey: failed to register hotkey: %v", err)
		return
	}

	log.Printf("Global hotkey: %v is registered\n", hk)

	// Listen for the hotkey press in a separate goroutine
	go func() {
		for {
			select {
			case <-hk.Keydown():
				log.Printf("Global hotkey: %v is down\n", hk)
				// Launch the game when hotkey is pressed
				go startGame(client)
			case <-hk.Keyup():
				log.Printf("Global hotkey: %v is up\n", hk)
			}
		}
	}()
}

// Function for handling tray menu and actions
func onReady(icon []byte, client *pepeunit.PepeunitClient) {
	// Set tray icon and menu options
	systray.SetIcon(icon)
	systray.SetTitle("Tray Example")
	systray.SetTooltip("Minimal Tray App")

	// Menu item to start the game
	mButton := systray.AddMenuItem("Меню", "Нажмите для выполнения")
	go func() {
		for {
			select {
			case <-mButton.ClickedCh:
				go startGame(client)
			}
		}
	}()

	// Exit menu item
	mQuit := systray.AddMenuItem("Выход", "Закрыть приложение")
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

// Exit handler for the system tray
func onExit() {
	fmt.Println("Приложение завершено")
}

// Function to load the icon image
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

// Function to prepare the game setup
func prepareGame(client *pepeunit.PepeunitClient) (*game.Game, error) {
	data, err := queries.GetUnitsByNodesQuery()
	if err != nil {
		return nil, fmt.Errorf("Ошибка при получении данных: %v", err)
	}

	// Update the game configuration
	config.UpdateConfig(func(cfg *config.Config) {
		cfg.BlurredBackground = graphics.BlurScreenshot()
	})
	// Load state from REST storage via pepeunit client
	stateData := make(map[string][][]string)
	if client.GetRESTClient() != nil {
		ctx := context.Background()
		raw, err := client.GetRESTClient().GetStateStorage(ctx, "")
		if err == nil && raw != nil {
			var stateStr string
			if s, ok := raw["state"].(string); ok {
				stateStr = s
			} else if s, ok := raw["State"].(string); ok {
				stateStr = s
			}
			if stateStr != "" && stateStr != "\"\"" {
				_ = json.Unmarshal([]byte(stateStr), &stateData)
			}
		}
	}

	return &game.Game{
		PepeClient:       client,
		Units:            data,
		SelectedSegments: make([]int, 3),
		KeyDownMap:       make(map[ebiten.Key]bool),
		StateData:        stateData,
		ActiveLayer:      0,
	}, nil
}

// Function to start the game
func startGame(client *pepeunit.PepeunitClient) {
	// Prepare the game
	gameInstance, err := prepareGame(client)
	if err != nil {
		log.Fatalf("Ошибка при подготовке игры: %v", err)
		return
	}

	// Start the game in fullscreen mode
	ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("Picker")

	// Run the game
	if err := ebiten.RunGame(gameInstance); err != nil {
		log.Printf("Ошибка запуска игры: %v", err)
	}

	restartApplication()
}

// Function to restart the application
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
