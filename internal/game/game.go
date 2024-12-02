package game

import (
	"image/color"
	"math"
    "fmt"
	"picker/internal/config"
	"picker/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
    "picker/internal/mqttclient"
    "picker/internal/queries"
    "picker/internal/state"
)

type Game struct{
    Client *mqttclient.MqttClient
    Units queries.UnitsByNodesResponse
    StateManager state.StateManager
    isMouseDown bool
    SelectSegment int
}

func (g *Game) Update() error {
    // Закрыть игру при нажатии клавиши Esc
    if ebiten.IsKeyPressed(ebiten.KeyEscape) {
        return fmt.Errorf("game closed by user")
    }

    cfg := config.GetConfig()
    mouseX, mouseY := ebiten.CursorPosition()
    dx, dy := mouseX-cfg.PickerCenterX, mouseY-cfg.PickerCenterY
    angle := math.Atan2(-float64(dy), -float64(dx)) + math.Pi

    segmentAngle := 2 * math.Pi / float64(g.Units.Count)

    // Обновляем выбранный сегмент
    config.UpdateConfig(func(cfg *config.Config) {
        g.SelectSegment = int(angle / segmentAngle) % g.Units.Count
    })

    // Обработка нажатия мыши
    if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
        if !g.isMouseDown {
            // Если кнопка была не нажата, а теперь нажата
            g.isMouseDown = true
            if g.Units.Count > 0 && g.SelectSegment == 1 {
                fmt.Println(g.Units.Units)
                err := g.Client.Publish("devunit.pepeunit.com/6d26314c-a030-498f-a5ef-b7544f460f88/pepeunit", 0, false, "{\"sleep\": 10, \"duty\": 32000}") 
                if err == nil {
                    fmt.Println("Sendet")
                }
            }
        }
    } else {
        // Сбрасываем флаг, если кнопка отпущена
        g.isMouseDown = false
    }

    return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	cfg := config.GetConfig()

	if cfg.BlurredBackground == nil {
		ebitenutil.DebugPrint(screen, "Загрузка размытого фона...")
		return
	}

	screen.DrawImage(cfg.BlurredBackground, nil)

	segmentAngle := 2 * math.Pi / float64(g.Units.Count)
	for i := 0; i < g.Units.Count; i++ {
		angleStart := float64(i) * segmentAngle
		angleEnd := angleStart + segmentAngle
		clr := color.RGBA{255, 255, 255, 128}
		if i == g.SelectSegment {
			clr = color.RGBA{255, 0, 0, 200}
		}
		graphics.DrawSegment(screen, cfg.PickerCenterX, cfg.PickerCenterY, cfg.RadiusInner, cfg.RadiusOuter, angleStart, angleEnd, clr)
	}

	if g.SelectSegment >= 0 {
		ebitenutil.DebugPrint(screen, "segment: " + g.Units.Units[g.SelectSegment].Name)
	}

}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	cfg := config.GetConfig()
	return cfg.ScreenWidth, cfg.ScreenHeight
}
