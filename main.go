package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"picker/internal/config"
	"picker/internal/game"
	"picker/internal/graphics"
	"picker/internal/queries"
	"bytes"
	"image"
	"image/png"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/getlantern/systray"
)

var gameRunning bool // Флаг для отслеживания состояния игры
var blurredBackground *ebiten.Image // Сохранение размытого фона заранее

//go:embed assets/icons/64.png
var iconData []byte

func main() {
	// Подготовка иконки и других ресурсов
	icon, err := loadIcon(iconData)
	if err != nil {
		log.Fatal("Ошибка загрузки иконки:", err)
	}

	// Загрузка размытого фона
	blurredBackground = graphics.BlurScreenshot()

	// Запускаем иконку в системном лотке
	systray.Run(func() {
		onReady(icon)
	}, onExit)
}

func onReady(icon []byte) {
	// Устанавливаем иконку
	systray.SetIcon(icon)

	// Заголовок и тултип
	systray.SetTitle("Tray Example")
	systray.SetTooltip("Minimal Tray App")

	// Кнопка для запуска игры
	mButton := systray.AddMenuItem("Выполнить действие", "Нажмите для выполнения")

	// Горутина для обработки нажатия на кнопку
	go func() {
		for {
			select {
			case <-mButton.ClickedCh:
				// Запускаем игру в горутине, если она не запущена
				if !gameRunning {
					gameRunning = true
					go startGame()
				} else {
					log.Println("Игра уже запущена.")
				}
			}
		}
	}()

	// Кнопка выхода
	mQuit := systray.AddMenuItem("Выход", "Закрыть приложение")
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

func onExit() {
	// Очистка ресурсов перед выходом
	fmt.Println("Приложение завершено")
}

// Функция для загрузки иконки
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

// Функция для подготовки игры
func prepareGame() (*game.Game, error) {
	// Получаем данные для настройки игры
	data, err := queries.GetUnitsByNodesQuery()
	if err != nil {
		return nil, fmt.Errorf("Ошибка при получении данных: %v", err)
	}

	// Обновляем конфигурацию с новыми данными
	config.UpdateConfig(func(cfg *config.Config) {
		cfg.BlurredBackground = blurredBackground // Используем заранее загруженный фон
		cfg.NumSegments = data.Count
	})

	// Создаем объект игры
	return &game.Game{}, nil
}

// Функция для запуска игры
func startGame() {
	// Подготовка игры
	gameInstance, err := prepareGame()
	if err != nil {
		log.Fatalf("Ошибка при подготовке игры: %v", err)
		return
	}

	// Запуск игры в полноэкранном режиме
	ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("Picker")

	// Проверяем, не запущена ли игра уже
	if err := ebiten.RunGame(gameInstance); err != nil {
		log.Printf("Ошибка запуска игры: %v", err)
	}

	// После завершения игры сбрасываем флаг
	gameRunning = false

	// Перезапускаем приложение
	restartApplication()
}

// Функция для перезапуска приложения
func restartApplication() {
	// Получаем текущий путь к исполнимому файлу
	exe, err := os.Executable()
	if err != nil {
		log.Fatal("Не удалось получить исполнимый файл:", err)
	}

	// Перезапускаем приложение
	cmd := exec.Command(exe)
	if err := cmd.Start(); err != nil {
		log.Fatal("Не удалось перезапустить приложение:", err)
	}

	// Завершаем текущий процесс
	os.Exit(0)
}

