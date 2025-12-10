package main

import (
	"bytes"
	"context"
	_ "embed"
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
	"strings"

	"time"

	"github.com/getlantern/systray"
	"github.com/hajimehoshi/ebiten/v2"
	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
)

//go:embed assets/icons/64.png
var iconData []byte

func main() {
	pepeClient, err := pepeunit.NewPepeunitClient(pepeunit.PepeunitClientConfig{
		EnvFilePath:      "env.json",
		SchemaFilePath:   "schema.json",
		LogFilePath:      "log.json",
		EnableMQTT:       true,
		EnableREST:       true,
		CycleSpeed:       100 * time.Millisecond,
		RestartMode:      pepeunit.RestartModeRestartExec,
		SkipVersionCheck: true,
	})
	if err != nil {
		log.Fatalf("init pepeunit client failed: %v", err)
	}

	log.Println("App starting...")

	ctx := context.Background()
	if pepeClient.GetMQTTClient() != nil {
		if err := pepeClient.GetMQTTClient().Connect(ctx); err != nil {
			log.Fatalf("mqtt connect failed: %v", err)
		}
		// Enable base MQTT handlers and subscribe to schema topics
		pepeClient.SetMQTTInputHandler(nil)
		if err := pepeClient.SubscribeAllSchemaTopics(ctx); err != nil {
			log.Printf("subscribe topics failed: %v", err)
		}
	}
	go pepeClient.RunMainCycle(ctx, nil)

	// Initialize the hotkey listener in the main thread
	mainthread.Init(func() {
		registerGlobalHotkey(pepeClient)
		registerOptionHotkeys(pepeClient)
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

// registerOptionHotkeys регистрирует глобальные хоткеи Ctrl+Shift+<буква> для опций,
// сохранённых в StateStorage (третье поле в массиве: [name, value, hotkey]).
func registerOptionHotkeys(client *pepeunit.PepeunitClient) {
	if client == nil {
		return
	}

	log.Println("Trying to register option hotkeys from state...")

	ctx := context.Background()
	stateData := make(map[string][][]string)

	if stateStr, err := client.GetStateStorage(ctx); err == nil && stateStr != "" && stateStr != "\"\"" {
		// Основной путь: состояние хранится как обычный JSON-объект
		if err := json.Unmarshal([]byte(stateStr), &stateData); err != nil {
			// Fallback: состояние может быть сохранено как JSON-строка внутри строки
			var wrapped string
			if err2 := json.Unmarshal([]byte(stateStr), &wrapped); err2 == nil && wrapped != "" {
				_ = json.Unmarshal([]byte(wrapped), &stateData)
			}
		}
	}

	if len(stateData) == 0 {
		log.Println("No state data found for option hotkeys")
		return
	}

	settings := client.GetSettings()

	type hotkeyBinding struct {
		topic   string
		payload string
	}

	// Глобальная уникальность по букве: одна буква — один биндинг.
	bindings := make(map[rune]hotkeyBinding)

	for nodeUUID, items := range stateData {
		for _, pair := range items {
			if len(pair) < 3 {
				continue
			}
			rawHotkey := strings.TrimSpace(pair[2])
			if rawHotkey == "" {
				continue
			}
			runes := []rune(strings.ToUpper(rawHotkey))
			if len(runes) != 1 {
				continue
			}
			ch := runes[0]
			if ch < 'A' || ch > 'Z' {
				continue
			}

			// Уже есть биндинг для этой буквы — пропускаем, т.к. внутри игры мы
			// уже гарантируем уникальность по букве.
			if _, exists := bindings[ch]; exists {
				continue
			}

			if len(pair) < 2 {
				continue
			}

			topicName := settings.PU_DOMAIN + "/" + nodeUUID + "/pepeunit"
			bindings[ch] = hotkeyBinding{
				topic:   topicName,
				payload: pair[1],
			}
		}
	}

	if len(bindings) == 0 {
		log.Println("No option hotkeys found in state")
		return
	}

	// Маппинг символа в hotkey.Key
	keyMap := map[rune]hotkey.Key{
		'A': hotkey.KeyA,
		'B': hotkey.KeyB,
		'C': hotkey.KeyC,
		'D': hotkey.KeyD,
		'E': hotkey.KeyE,
		'F': hotkey.KeyF,
		'G': hotkey.KeyG,
		'H': hotkey.KeyH,
		'I': hotkey.KeyI,
		'J': hotkey.KeyJ,
		'K': hotkey.KeyK,
		'L': hotkey.KeyL,
		'M': hotkey.KeyM,
		'N': hotkey.KeyN,
		'O': hotkey.KeyO,
		'P': hotkey.KeyP,
		'Q': hotkey.KeyQ,
		'R': hotkey.KeyR,
		'S': hotkey.KeyS,
		'T': hotkey.KeyT,
		'U': hotkey.KeyU,
		'V': hotkey.KeyV,
		'W': hotkey.KeyW,
		'X': hotkey.KeyX,
		'Y': hotkey.KeyY,
		'Z': hotkey.KeyZ,
	}

	for ch, bind := range bindings {
		keyConst, ok := keyMap[ch]
		if !ok {
			continue
		}

		hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}, keyConst)

		if err := hk.Register(); err != nil {
			log.Printf("hotkey: failed to register option hotkey Ctrl+Shift+%c: %v", ch, err)
			continue
		}

		log.Printf("Option hotkey Ctrl+Shift+%c is registered\n", ch)

		// Слушаем нажатие хоткея в отдельной горутине
		go func(hk *hotkey.Hotkey, bind hotkeyBinding, ch rune) {
			for {
				select {
				case <-hk.Keydown():
					log.Printf("Option hotkey Ctrl+Shift+%c is down, publishing to %s\n", ch, bind.topic)
					if client != nil && client.GetMQTTClient() != nil {
						if err := client.GetMQTTClient().Publish(bind.topic, bind.payload); err != nil {
							log.Printf("failed to publish MQTT message for hotkey %c: %v", ch, err)
						}
					}
				case <-hk.Keyup():
					log.Printf("Option hotkey Ctrl+Shift+%c is up\n", ch)
				}
			}
		}(hk, bind, ch)
	}
}

// Register the global hotkey (Ctrl + Shift + H)
func registerGlobalHotkey(client *pepeunit.PepeunitClient) {
	log.Println("Trying to register global hotkey...")

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
	log.Println("systray onReady called")

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
	data, err := game.FetchUnits(client)
	if err != nil {
		// Ошибка загрузки юнитов не блокирует запуск игры:
		// просто логируем и продолжаем с пустым списком Units.
		log.Printf("Ошибка при получении данных (используем пустой список юнитов): %v", err)
	}

	// Update the game configuration
	config.UpdateConfig(func(cfg *config.Config) {
		cfg.BlurredBackground = graphics.BlurScreenshot()
	})
	// Load state from Pepeunit storage via high-level client API
	stateData := make(map[string][][]string)
	ctx := context.Background()
	if stateStr, err := client.GetStateStorage(ctx); err == nil && stateStr != "" && stateStr != "\"\"" {
		// Основной путь: состояние хранится как обычный JSON-объект
		if err := json.Unmarshal([]byte(stateStr), &stateData); err != nil {
			// Fallback: состояние может быть сохранено как JSON-строка внутри строки
			var wrapped string
			if err2 := json.Unmarshal([]byte(stateStr), &wrapped); err2 == nil && wrapped != "" {
				_ = json.Unmarshal([]byte(wrapped), &stateData)
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
