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
    ActiveLayer   int // Индекс текущего слоя
    Layers        [][]string // Слои, где каждый содержит свои элементы
}

func (g *Game) Update() error {
    if ebiten.IsKeyPressed(ebiten.KeyEscape) {
        return fmt.Errorf("game closed by user")
    }

    cfg := config.GetConfig()
    mouseX, mouseY := ebiten.CursorPosition()
    dx, dy := mouseX-cfg.PickerCenterX, mouseY-cfg.PickerCenterY
    angle := math.Atan2(-float64(dy), -float64(dx)) + math.Pi

    currentLayer := g.Layers[g.ActiveLayer]
    segmentAngle := 2 * math.Pi / float64(len(currentLayer))

    g.SelectSegment = int(angle / segmentAngle) % len(currentLayer)

    if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
        if !g.isMouseDown {
            g.isMouseDown = true
            // Открываем следующий слой, если он есть
            if g.ActiveLayer < len(g.Layers)-1 {
                g.ActiveLayer++
                g.SelectSegment = 0 // Сброс выбранного сегмента
            } else {
                fmt.Println("Активный элемент:", currentLayer[g.SelectSegment])
                // Логика взаимодействия с последним слоем
            }
        }
    } else if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
        if !g.isMouseDown {
            g.isMouseDown = true
            // Возвращаемся на предыдущий слой, если он есть
            if g.ActiveLayer > 0 {
                g.ActiveLayer--
                g.SelectSegment = 0 // Сброс выбранного сегмента
            }
        }
    } else {
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

    // Отрисовываем все слои от 0 до ActiveLayer включительно
    for layerIndex := 0; layerIndex <= g.ActiveLayer; layerIndex++ {
        currentLayer := g.Layers[layerIndex]
        segmentAngle := 2 * math.Pi / float64(len(currentLayer))
        layerOffset := float64(layerIndex) * 60 // Смещение радиуса для каждого слоя

        for i := 0; i < len(currentLayer); i++ {
            angleStart := float64(i) * segmentAngle
            angleEnd := angleStart + segmentAngle
            clr := color.RGBA{176, 190, 197, 255}
            if layerIndex == g.ActiveLayer && i == g.SelectSegment {
                // Выделяем текущий элемент только на активном слое
                clr = color.RGBA{255, 61, 0, 255}
            }
            graphics.DrawSegment(
                screen,
                cfg.PickerCenterX,
                cfg.PickerCenterY,
                cfg.RadiusInner+ int(layerOffset),
                cfg.RadiusInner+int(layerOffset)+cfg.ThickSegment,
                angleStart+0.01,
                angleEnd-0.01,
                clr,
            )
        }
    }

    // Показываем название выбранного элемента только для активного слоя
    if g.SelectSegment >= 0 && g.ActiveLayer < len(g.Layers) {
        currentLayer := g.Layers[g.ActiveLayer]
        ebitenutil.DebugPrintAt(
            screen,
            currentLayer[g.SelectSegment],
            int(cfg.ScreenWidth/2),
            int(cfg.ScreenHeight/2),
        )
    }
}


func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	cfg := config.GetConfig()
	return cfg.ScreenWidth, cfg.ScreenHeight
}
