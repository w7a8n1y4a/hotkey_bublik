package game

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"image/color"
	"math"
	"picker/internal/config"
	"picker/internal/graphics"
	"picker/internal/mqttclient"
	"picker/internal/queries"
	"picker/internal/state"
)

type Game struct {
	Client        *mqttclient.MqttClient
	Units         queries.UnitsByNodesResponse
	StateManager  state.StateManager
	KeyDownMap    map[ebiten.Key]bool // Состояние кнопок
	SelectSegment int
	ActiveLayer   int // Индекс текущего слоя
}

func (g *Game) Update() error {
    if ebiten.IsKeyPressed(ebiten.KeyEscape) {
        return fmt.Errorf("game closed by user")
    }

    cfg := config.GetConfig()
    mouseX, mouseY := ebiten.CursorPosition()
    dx, dy := mouseX-cfg.PickerCenterX, mouseY-cfg.PickerCenterY
    angle := math.Atan2(-float64(dy), -float64(dx)) + math.Pi

    var currentLayerLength int
    switch g.ActiveLayer {
    case 0: // Первый слой — список Units
        currentLayerLength = len(g.Units.Units)
    case 1: // Второй слой — UnitNodes выбранного Unit
        if g.SelectSegment < len(g.Units.Units) {
            currentLayerLength = len(g.Units.Units[g.SelectSegment].UnitNodes)
        }
    case 2: // Третий слой — данные из StateManager
        if g.SelectSegment < len(g.Units.Units) {
            selectedUnit := g.Units.Units[g.SelectSegment]
            if g.SelectSegment < len(selectedUnit.UnitNodes) {
                selectedNode := selectedUnit.UnitNodes[g.SelectSegment]
                stateData := g.StateManager.GetState()[selectedNode.UUID]
                currentLayerLength = len(stateData)
            }
        }
    }

    if currentLayerLength > 0 {
        segmentAngle := 2 * math.Pi / float64(currentLayerLength)
        g.SelectSegment = int(angle / segmentAngle) % currentLayerLength
    }

    // Обработка нажатий
    g.handleKey(ebiten.KeyDelete, func() {
        // Удаление работает только на третьем слое
        if g.ActiveLayer == 2 {
            if g.SelectSegment < len(g.Units.Units) {
                selectedUnit := g.Units.Units[g.SelectSegment]
                if g.SelectSegment < len(selectedUnit.UnitNodes) {
                    selectedNode := selectedUnit.UnitNodes[g.SelectSegment]
                    stateData := g.StateManager.GetState()[selectedNode.UUID]
                    keys := make([]string, 0, len(stateData))
                    for key := range stateData {
                        keys = append(keys, key)
                    }
                    if g.SelectSegment < len(keys) {
                        delete(g.StateManager.GetState()[selectedNode.UUID], keys[g.SelectSegment])
                    }
                }
            }
        }
    })

    g.handleKey(ebiten.Key(ebiten.MouseButtonLeft), func() {
        // Обработка левой кнопки мыши
        if g.ActiveLayer < 2 {
            g.ActiveLayer++
            g.SelectSegment = 0 // Сброс выбора на следующем слое
        } else {
            // Логика добавления нового элемента
            if g.ActiveLayer == 2 {
                if g.SelectSegment < len(g.Units.Units) {
                    selectedUnit := g.Units.Units[g.SelectSegment]
                    if g.SelectSegment < len(selectedUnit.UnitNodes) {
                        selectedNode := selectedUnit.UnitNodes[g.SelectSegment]
                        stateData := g.StateManager.GetState()[selectedNode.UUID]
                        stateData[fmt.Sprintf("New Key %d", len(stateData)+1)] = "New Value"
                    }
                }
            }
        }
    })

    g.handleKey(ebiten.Key(ebiten.MouseButtonRight), func() {
        // Обработка правой кнопки мыши
        if g.ActiveLayer > 0 {
            g.ActiveLayer--
            g.SelectSegment = 0 // Сброс выбора на предыдущем слое
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
		var items []string

		switch layerIndex {
		case 0: // Первый слой — Units
			for _, unit := range g.Units.Units {
				items = append(items, unit.Name)
			}
		case 1: // Второй слой — UnitNodes
			if g.SelectSegment < len(g.Units.Units) {
				for _, node := range g.Units.Units[g.SelectSegment].UnitNodes {
					items = append(items, node.TopicName)
				}
			}
		case 2: // Третий слой — данные из StateManager
			if g.SelectSegment < len(g.Units.Units) {
				selectedUnit := g.Units.Units[g.SelectSegment]
				if g.SelectSegment < len(selectedUnit.UnitNodes) {
					selectedNode := selectedUnit.UnitNodes[g.SelectSegment]
					stateData := g.StateManager.GetState()[selectedNode.UUID]
                    
                    items = append(items, "New Item")

					for key, value := range stateData {
						items = append(items, fmt.Sprintf("%s: %s", key, value))
					}
				}
			}
		}

		// Рисуем сегменты текущего слоя
		segmentAngle := 2 * math.Pi / float64(len(items))
		layerOffset := float64(layerIndex) * 60
		for i, _ := range items {
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
        }
        if g.SelectSegment >= 0 {
            ebitenutil.DebugPrintAt(
                screen,
                items[g.SelectSegment],
                int(cfg.ScreenWidth/2),
                int(cfg.ScreenHeight/2) + 10 * layerIndex,
            )
        }

	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	cfg := config.GetConfig()
	return cfg.ScreenWidth, cfg.ScreenHeight
}
