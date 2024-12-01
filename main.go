package main

import (
    _ "embed"
	"log"
    "fmt"
    "image"
    "image/png"
    "bytes"
	"picker/internal/config"
	"picker/internal/game"
	"picker/internal/graphics"
    "picker/internal/queries"

	"github.com/hajimehoshi/ebiten/v2"

    "github.com/getlantern/systray"
    "os/exec"
)

//go:embed assets/icons/64.png
var iconData []byte

func main() {
    systray.Run(onReady, onExit)

    data, err := queries.GetUnitsByNodesQuery()
    fmt.Println(data, err)

	// // Вывод результата
	// fmt.Printf("Общее количество: %d\n", data.Count)
	// for _, unit := range data.Units {
	// 	fmt.Printf("UUID: %s, Name: %s, Create Date: %s\n", unit.UUID, unit.Name, unit.CreateDatetime)
	// 	for _, node := range unit.UnitNodes {
	// 		fmt.Printf("\tNode UUID: %s, Type: %s, Topic: %s\n", node.UUID, node.Type, node.TopicName)
	// 	}
	// }
    
    config.UpdateConfig(func(cfg *config.Config) {
        cfg.BlurredBackground = graphics.BlurScreenshot()
        cfg.NumSegments = data.Count
    })
    


    ebiten.SetFullscreen(true)
    ebiten.SetWindowTitle("Picker")
    if err := ebiten.RunGame(&game.Game{}); err != nil {
        log.Fatalf("Ошибка запуска игры: %v", err)
    }

    select{}

}

// Функция для загрузки иконки из файла PNG
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

func onReady() {
    icon, err := loadIcon(iconData)
    if err != nil {
        log.Fatal("Ошибка загрузки иконки:", err)
    }

	// Загружаем иконку
	systray.SetIcon(icon)

	// Устанавливаем текст для иконки
	systray.SetTitle("Tray Example")
	systray.SetTooltip("Minimal Tray App")

	// Создаём кнопку
	mButton := systray.AddMenuItem("Выполнить действие", "Нажмите для выполнения")

	// Горутина для обработки нажатия на кнопку
	go func() {
		for {
			select {
			case <-mButton.ClickedCh:
				// Действие при нажатии кнопки
				fmt.Println("Кнопка нажата!")
				// Например, откроем терминал
				exec.Command("xterm").Start()
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

