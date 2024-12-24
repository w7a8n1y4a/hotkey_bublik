package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"os/exec"
	"picker/internal/config"
	"picker/internal/game"
	"picker/internal/graphics"
	"picker/internal/mqttclient"
	"picker/internal/queries"
	"picker/internal/state"

	"github.com/getlantern/systray"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
)

// Global variables
var gameRunning bool                // Flag to track the game state
var blurredBackground *ebiten.Image // To store the blurred background

//go:embed assets/icons/64.png
var iconData []byte

func main() {

    client, err := mqttclient.RunMqttClient()
	fmt.Println(client, err)

	// Initialize the hotkey listener in the main thread
	mainthread.Init(func() {
		registerGlobalHotkey(client)
	})

	// Prepare icon and other resources
	icon, err := loadIcon(iconData)
	if err != nil {
		log.Fatal("Ошибка загрузки иконки:", err)
	}

	// Load the blurred background
	blurredBackground = graphics.BlurScreenshot()

	// Start the system tray app
	systray.Run(func() {
		onReady(icon, client)
	}, onExit)
}

// Register the global hotkey (Ctrl + Shift + H)
func registerGlobalHotkey(client *mqttclient.MqttClient) {
	hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}, hotkey.KeyH)

	err := hk.Unregister()
	if err != nil {
		log.Printf("hotkey: failed to unregister previous hotkey: %v", err)
	}

	err = hk.Register()
	if err != nil {
		log.Fatalf("hotkey: failed to register hotkey: %v", err)
		return
	}

	log.Printf("Global hotkey: %v is registered\n", hk)
	// Listen for the hotkey press in a separate goroutine
	go func() {
		<-hk.Keydown()
		log.Printf("Global hotkey: %v is down\n", hk)
		// Launch the game when hotkey is pressed
		if !gameRunning {
			gameRunning = true
			go startGame(client)
		}
		<-hk.Keyup()
		log.Printf("Global hotkey: %v is up\n", hk)
	}()
}

// Function for handling tray menu and actions
func onReady(icon []byte, client *mqttclient.MqttClient) {
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
				if !gameRunning {
					gameRunning = true
					go startGame(client)
				} else {
					log.Println("Игра уже запущена.")
				}
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
func prepareGame(client *mqttclient.MqttClient) (*game.Game, error) {
	data, err := queries.GetUnitsByNodesQuery()
	if err != nil {
		return nil, fmt.Errorf("Ошибка при получении данных: %v", err)
	}

	// Update the game configuration
	config.UpdateConfig(func(cfg *config.Config) {
		cfg.BlurredBackground = blurredBackground
	})
    stateAppManager, err := state.NewStateManager()
    if err != nil {
		return nil, fmt.Errorf("Ошибка полученя state: %v", err)
	}

    return &game.Game{
        Client: client,
        Units: data,
        SelectedSegments: make([]int, 3), 
        KeyDownMap: make(map[ebiten.Key]bool),
        StateManager: stateAppManager,
        ActiveLayer: 0,}, nil
}

// Function to start the game
func startGame(client *mqttclient.MqttClient) {
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

	// After game finishes, reset the flag
	gameRunning = false
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
