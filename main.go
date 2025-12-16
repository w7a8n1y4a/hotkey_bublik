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
	"picker/internal/hotkeys"
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
		mods    []hotkey.Modifier
		key     hotkey.Key
	}

	// Глобальная уникальность по строке хоткея: один хоткей — один биндинг.
	bindings := make(map[string]hotkeyBinding)

	for nodeUUID, items := range stateData {
		for _, pair := range items {
			if len(pair) < 3 {
				continue
			}
			rawHotkey := strings.TrimSpace(pair[2])
			if rawHotkey == "" {
				continue
			}

			// Парсим хоткей для валидации и нормализации
			mods, key, display, err := hotkeys.ParseHotkeySpec(rawHotkey)
			if err != nil {
				log.Printf("hotkey: invalid hotkey spec %q: %v; skipping", rawHotkey, err)
				continue
			}

			// Используем нормализованную строку как ключ для уникальности
			if _, exists := bindings[display]; exists {
				continue
			}

			if len(pair) < 2 {
				continue
			}

			topicName := settings.PU_DOMAIN + "/" + nodeUUID + "/pepeunit"
			bindings[display] = hotkeyBinding{
				topic:   topicName,
				payload: pair[1],
				mods:    mods,
				key:     key,
			}
		}
	}

	if len(bindings) == 0 {
		log.Println("No option hotkeys found in state")
		return
	}

	for display, bind := range bindings {
		hk := hotkey.New(bind.mods, bind.key)

		if err := hk.Register(); err != nil {
			log.Printf("hotkey: failed to register option hotkey %s: %v", display, err)
			continue
		}

		log.Printf("Option hotkey %s is registered\n", display)

		// Слушаем нажатие хоткея в отдельной горутине
		go func(hk *hotkey.Hotkey, bind hotkeyBinding, display string) {
			for {
				select {
				case <-hk.Keydown():
					log.Printf("Option hotkey %s is down, publishing to %s\n", display, bind.topic)
					if client != nil && client.GetMQTTClient() != nil {
						if err := client.GetMQTTClient().Publish(bind.topic, bind.payload); err != nil {
							log.Printf("failed to publish MQTT message for hotkey %s: %v", display, err)
						}
					}
				case <-hk.Keyup():
					log.Printf("Option hotkey %s is up\n", display)
				}
			}
		}(hk, bind, display)
	}
}

// registerGlobalHotkey регистрирует глобальный хоткей запуска интерфейса,
// который задаётся только через HOTKEY_MAIN (без дефолта в коде).
func registerGlobalHotkey(client *pepeunit.PepeunitClient) {
	cfg := config.GetConfig()
	if cfg.LaunchHotkeyMain == nil {
		log.Println("Global hotkey is disabled (HOTKEY_MAIN is null/empty/missing)")
		return
	}

	mods, key, display, err := hotkeys.ParseHotkeySpec(*cfg.LaunchHotkeyMain)
	if err != nil {
		log.Printf("hotkey: invalid HOTKEY_MAIN=%q: %v; global hotkey is disabled", *cfg.LaunchHotkeyMain, err)
		return
	}

	log.Printf("Trying to register global hotkey %s...", display)
	hk := hotkey.New(mods, key)

	if err := hk.Register(); err != nil {
		// Не падаем: хоткей может быть занят или запрещён окружением.
		log.Printf("hotkey: failed to register global hotkey %s: %v", display, err)
		return
	}

	log.Printf("Global hotkey %s is registered\n", display)

	// Listen for the hotkey press in a separate goroutine
	go func() {
		for {
			select {
			case <-hk.Keydown():
				log.Printf("Global hotkey %s is down\n", display)
				go startGame(client)
			case <-hk.Keyup():
				log.Printf("Global hotkey %s is up\n", display)
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
		for range mButton.ClickedCh {
			go startGame(client)
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

	return game.NewGame(client, data, stateData)
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
