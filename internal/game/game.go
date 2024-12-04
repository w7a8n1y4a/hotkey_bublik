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

type Game struct {
    Client       *mqttclient.MqttClient
    Units        queries.UnitsByNodesResponse
    StateManager state.StateManager
    KeyDownMap   map[ebiten.Key]bool // Состояние кнопок
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

    g.handleKey(ebiten.KeyDelete, func() {
        if g.ActiveLayer == 2 { // Удаление работает только на третьем слое
            selectedItem := currentLayer[g.SelectSegment]
            if selectedItem != "Add New" {
                // Удаляем выбранный элемент
                g.Layers[g.ActiveLayer] = append(
                    g.Layers[g.ActiveLayer][:g.SelectSegment],
                    g.Layers[g.ActiveLayer][g.SelectSegment+1:]...,
                )
                // Перемещаем выбор на предыдущий сегмент, если он есть
                if g.SelectSegment >= len(g.Layers[g.ActiveLayer]) {
                    g.SelectSegment = len(g.Layers[g.ActiveLayer]) - 1
                }
            }
        }
    })

    g.handleKey(ebiten.Key(ebiten.MouseButtonLeft), func() {
        // Обработка левой кнопки мыши
        if g.ActiveLayer < len(g.Layers)-1 {
            g.ActiveLayer++
            g.SelectSegment = 0 // Сброс выбранного сегмента
        } else {
            selectedItem := currentLayer[g.SelectSegment]
            if g.ActiveLayer == 2 && selectedItem == "Add New" {
                // Логика добавления нового сегмента
                g.Layers[g.ActiveLayer] = append(g.Layers[g.ActiveLayer], fmt.Sprintf("New %d", len(currentLayer)))
            } else {
                fmt.Println("Активный элемент:", selectedItem)
            }
        }
    })

    g.handleKey(ebiten.Key(ebiten.MouseButtonRight), func() {
        // Обработка правой кнопки мыши
        if g.ActiveLayer > 0 {
            g.ActiveLayer--
            g.SelectSegment = 0 // Сброс выбранного сегмента
        }
    })

    return nil
}

// handleKey - обобщённая обработка для любых кнопок
func (g *Game) handleKey(key ebiten.Key, action func()) {
    // Пытаемся использовать для мыши одинаковую логику
    keyPressed := false
    if key == ebiten.Key(ebiten.MouseButtonLeft) {
        keyPressed = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
    } else if key == ebiten.Key(ebiten.MouseButtonRight) {
        keyPressed = ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
    } else {
        keyPressed = ebiten.IsKeyPressed(key)
    }

    if keyPressed {
        if !g.KeyDownMap[key] {
            g.KeyDownMap[key] = true
            action() // выполняем действие при первом нажатии
        }
    } else {
        g.KeyDownMap[key] = false // сбрасываем состояние, если кнопка отпущена
    }
}


func (g *Game) Draw(screen *ebiten.Image) {
    cfg := config.GetConfig()

    if cfg.BlurredBackground == nil {
        ebitenutil.DebugPrint(screen, "Загрузка размытого фона...")
        return
    }

    screen.DrawImage(cfg.BlurredBackground, nil)

    for layerIndex := 0; layerIndex <= g.ActiveLayer; layerIndex++ {
        currentLayer := g.Layers[layerIndex]
        segmentAngle := 2 * math.Pi / float64(len(currentLayer))
        layerOffset := float64(layerIndex) * 60 // Смещение радиуса для каждого слоя

        for i := 0; i < len(currentLayer); i++ {
            angleStart := float64(i) * segmentAngle
            angleEnd := angleStart + segmentAngle
            clr := color.RGBA{176, 190, 197, 255}
            if layerIndex == g.ActiveLayer && i == g.SelectSegment {
                clr = color.RGBA{255, 61, 0, 255}
            }
            graphics.DrawSegment(
                screen,
                cfg.PickerCenterX,
                cfg.PickerCenterY,
                cfg.RadiusInner+int(layerOffset),
                cfg.RadiusInner+int(layerOffset)+cfg.ThickSegment,
                angleStart+0.01,
                angleEnd-0.01,
                clr,
            )

            if layerIndex == 2 && currentLayer[i] == "Add New" {
                textX := cfg.PickerCenterX + int(float64(cfg.RadiusInner+int(layerOffset)+cfg.ThickSegment/2)*math.Cos(angleStart+segmentAngle/2)) - 10
                textY := cfg.PickerCenterY - int(float64(cfg.RadiusInner+int(layerOffset)+cfg.ThickSegment/2)*math.Sin(angleStart+segmentAngle/2)) - 10
                ebitenutil.DebugPrintAt(screen, "+", textX, textY)
            }
        }
    }

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

