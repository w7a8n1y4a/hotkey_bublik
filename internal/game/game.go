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
	Client          *mqttclient.MqttClient
	Units           queries.UnitsByNodesResponse
	StateManager    *state.StateManager
	KeyDownMap      map[ebiten.Key]bool // Состояние кнопок
	SelectedSegments []int              // Хранение текущего выбора для каждого слоя
	ActiveLayer     int                 // Индекс текущего слоя
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
		if g.SelectedSegments[0] < len(g.Units.Units) {
			currentLayerLength = len(g.Units.Units[g.SelectedSegments[0]].UnitNodes)
		}
	case 2: // Третий слой — данные из StateManager
		if g.SelectedSegments[0] < len(g.Units.Units) {
			selectedUnit := g.Units.Units[g.SelectedSegments[0]]
			if g.SelectedSegments[1] < len(selectedUnit.UnitNodes) {
				selectedNode := selectedUnit.UnitNodes[g.SelectedSegments[1]]
				stateData := g.StateManager.GetState()[selectedNode.UUID]
				currentLayerLength = len(stateData) + 1
			}
		}
	}

	if currentLayerLength > 0 {
		segmentAngle := 2 * math.Pi / float64(currentLayerLength)
		g.SelectedSegments[g.ActiveLayer] = int(angle / segmentAngle) % currentLayerLength
	}

	g.handleKey(ebiten.KeyDelete, func() {
		if g.ActiveLayer == 2 {
			selectedUnitIdx := g.SelectedSegments[0]
			selectedNodeIdx := g.SelectedSegments[1]
			if selectedUnitIdx < len(g.Units.Units) {
				selectedUnit := g.Units.Units[selectedUnitIdx]
				if selectedNodeIdx < len(selectedUnit.UnitNodes) {
					selectedNode := selectedUnit.UnitNodes[selectedNodeIdx]
					stateData := g.StateManager.GetState()[selectedNode.UUID]
					keys := make([]string, 0, len(stateData))
					for key := range stateData {
						keys = append(keys, key)
					}
					if g.SelectedSegments[2] < len(keys) {
						delete(stateData, keys[g.SelectedSegments[2]])
					}
				}
			}
		}
	})

	g.handleKey(ebiten.Key(ebiten.MouseButtonLeft), func() {
		if g.ActiveLayer < 2 {
			g.ActiveLayer++
			g.SelectedSegments[g.ActiveLayer] = 0
		} else if g.ActiveLayer == 2 {
			selectedUnitIdx := g.SelectedSegments[0]
			selectedNodeIdx := g.SelectedSegments[1]
			if selectedUnitIdx < len(g.Units.Units) {
				selectedUnit := g.Units.Units[selectedUnitIdx]
				if selectedNodeIdx < len(selectedUnit.UnitNodes) {
					selectedNode := selectedUnit.UnitNodes[selectedNodeIdx]
                    fmt.Println(selectedNode.TopicName)
					stateData := g.StateManager.GetState()[selectedNode.UUID]
                    if g.SelectedSegments[2] == 0 {
                        fmt.Println("This is Add button")

                        g.StateManager.AddOption(
                            selectedNode.UUID,
                            fmt.Sprintf("Explosive %d", len(stateData) + 1),
                            "1",
                        )
                    } else {
                        if stateData != nil{
                        
                            fmt.Println(stateData)
                        }
                        
                    }
				}
			}
		}
	})

	g.handleKey(ebiten.Key(ebiten.MouseButtonRight), func() {
		if g.ActiveLayer > 0 {
			g.ActiveLayer--
			g.SelectedSegments[g.ActiveLayer] = 0
		}
	})

	return nil
}

func (g *Game) handleKey(key ebiten.Key, action func()) {
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
			action()
		}
	} else {
		g.KeyDownMap[key] = false
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
		case 0:
			for _, unit := range g.Units.Units {
				items = append(items, unit.Name)
			}
		case 1:
			if g.SelectedSegments[0] < len(g.Units.Units) {
				for _, node := range g.Units.Units[g.SelectedSegments[0]].UnitNodes {
					items = append(items, node.TopicName)
				}
			}
		case 2:
			if g.SelectedSegments[0] < len(g.Units.Units) {
				selectedUnit := g.Units.Units[g.SelectedSegments[0]]
				if g.SelectedSegments[1] < len(selectedUnit.UnitNodes) {
					selectedNode := selectedUnit.UnitNodes[g.SelectedSegments[1]]
					stateData := g.StateManager.GetState()[selectedNode.UUID]
                    
                    items = append(items, "New Item")

					for key, value := range stateData {
						items = append(items, fmt.Sprintf("%s: %s", key, value))
                        fmt.Println(key)
					}
                    fmt.Println("")
                    
				}
			}
		}

		segmentAngle := 2 * math.Pi / float64(len(items))
		layerOffset := float64(layerIndex) * 60
		for i := range items {
			angleStart := float64(i) * segmentAngle
			angleEnd := angleStart + segmentAngle
			clr := color.RGBA{176, 190, 197, 255}
			if i == g.SelectedSegments[layerIndex] {
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
		if g.SelectedSegments[layerIndex] >= 0 && len(items) > g.SelectedSegments[layerIndex] {
            ebitenutil.DebugPrintAt(
				screen,
				items[g.SelectedSegments[layerIndex]],
				int(cfg.ScreenWidth/2),
				int(cfg.ScreenHeight/2)+10*layerIndex,
			)
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	cfg := config.GetConfig()
	return cfg.ScreenWidth, cfg.ScreenHeight
}

