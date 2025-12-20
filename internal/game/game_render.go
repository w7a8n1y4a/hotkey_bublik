package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"picker/internal/config"
	"picker/internal/graphics"
)

func (g *Game) Draw(screen *ebiten.Image) {

	cfg := config.GetConfig()
	if cfg.BlurredBackground == nil {
		g.drawBlurLoadingMessage(screen)
		return
	}
	screen.DrawImage(cfg.BlurredBackground, nil)

	switch g.InputMode {
	case ModeGame:
		for layerIndex := 0; layerIndex <= g.ActiveLayer; layerIndex++ {
			var items [][]string

			switch layerIndex {
			case 0:
				items = append(items, []string{"Обновить список юнитов"})
				for _, unit := range g.Units.Units {
					items = append(items, []string{unit.Name})
				}
			case 1:
				unitIdx := g.SelectedSegments[0] - 1
				if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
					for _, node := range g.Units.Units[unitIdx].UnitNodes {
						items = append(items, []string{node.TopicName})
					}
				}
			case 2:
				unitIdx := g.SelectedSegments[0] - 1
				if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
					selectedUnit := g.Units.Units[unitIdx]
					if g.SelectedSegments[1] < len(selectedUnit.UnitNodes) {
						selectedNode := selectedUnit.UnitNodes[g.SelectedSegments[1]]
						stateData := g.StateData[selectedNode.UUID]

						items = append(items, []string{"Создание команды"})

						items = append(items, stateData...)
					}
				}
			}

			if len(items) == 0 {
				continue
			}

			segmentAngle := 2 * math.Pi / float64(len(items))
			layerOffset := float64(layerIndex) * 60

			selectedUnitColor := defaultSegmentColor
			unitIdx := g.SelectedSegments[0] - 1
			if unitIdx >= 0 && unitIdx < len(g.Units.Units) && len(unitColors) > 0 {
				selectedUnitColor = unitColors[unitIdx%len(unitColors)]
			}

			for i := range items {
				angleStart := float64(i) * segmentAngle
				angleEnd := angleStart + segmentAngle

				var clr color.Color = defaultSegmentColor

				switch layerIndex {
				case 0:
					clr = defaultSegmentColor
					if i == 0 && i == g.SelectedSegments[0] {
						clr = refreshSegmentColor
					} else if i > 0 && i == g.SelectedSegments[0] && len(unitColors) > 0 {
						uIdx := (i - 1) % len(unitColors)
						if uIdx < 0 {
							uIdx = 0
						}
						clr = unitColors[uIdx]
					}
				case 1, 2:
					clr = defaultSegmentColor
					if i == g.SelectedSegments[layerIndex] {
						clr = selectedUnitColor
					}
				}

				graphics.DrawSegment(
					screen,
					cfg.PickerCenterX,
					cfg.PickerCenterY,
					cfg.RadiusInner+int(layerOffset),
					cfg.RadiusInner+int(layerOffset)+cfg.ThickSegment,
					angleStart+0.01-0.0015*float64(layerIndex),
					angleEnd-0.01+0.0015*float64(layerIndex),
					clr,
				)
			}
			g.drawGameModeMessages(screen, layerIndex, items)
		}
	case ModeTextInput:
		g.drawTextInputMessages(screen)
	case ModeHotkeyInput:
		g.drawHotkeyInputMessages(screen)
	}

	g.drawSpinner(screen)

}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	cfg := config.GetConfig()
	return cfg.ScreenWidth, cfg.ScreenHeight
}
