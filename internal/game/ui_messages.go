package game

import (
	"encoding/json"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"

	"picker/internal/config"
	"picker/internal/hotkeys"
)

func (g *Game) drawBlurLoadingMessage(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "Загрузка размытого фона...")
}

func (g *Game) drawGameModeMessages(screen *ebiten.Image, layerIndex int, items [][]string) {
	if len(items) == 0 {
		return
	}

	if layerIndex != g.ActiveLayer {
		return
	}

	if g.SelectedSegments[layerIndex] < 0 || len(items) <= g.SelectedSegments[layerIndex] {
		return
	}

	cfg := config.GetConfig()

	fontSize := 24
	fontFace := LoadFont(float64(fontSize))

	segmentFontSize := 32
	segmentFontFace := LoadFont(float64(segmentFontSize))

	centerX := int(cfg.ScreenWidth / 2)
	valueColumnCenterX := int(cfg.ScreenWidth / 5) // вертикальная линия центра левой колонки значения (1/5 экрана)
	centerUnit := int(cfg.ScreenHeight/2) - int(float64(fontSize)/2)
	centerUnitNode := int(cfg.ScreenHeight/2) + int(float64(fontSize)*1.5)
	centerOption := int(cfg.ScreenHeight / 2)

	optionExternalLen := int(float64(cfg.RadiusInner) + float64(cfg.ThickSegment)*3 + float64(fontSize)*float64(layerIndex))

	var centerY int

	switch layerIndex {
	case 0:
		centerY = centerUnit
	case 1:
		centerY = centerUnitNode
	case 2:
		centerY = centerOption - optionExternalLen + fontSize
	}

	selectedSegmentText := items[g.SelectedSegments[layerIndex]][0]
	maxWidth := cfg.RadiusInner * 2 // Увеличиваем ширину для размещения большего количества символов
	if maxWidth < 50*12 {           // 50 символов примерно по 12 пикселей каждый
		maxWidth = 50 * 12
	}

	activeLayerOuterRadius := cfg.RadiusInner + layerIndex*60 + cfg.ThickSegment
	segmentLabelY := cfg.PickerCenterY - activeLayerOuterRadius - 10 // 10 пикселей выше внешнего края активного слоя (значительно ниже)

	DrawCenteredText(
		screen,
		segmentFontFace,
		selectedSegmentText,
		centerX,
		segmentLabelY,
		maxWidth,
		4,
		color.White,
	)

	if layerIndex == g.ActiveLayer && g.ActiveLayer >= 1 {
		unitIdx := g.SelectedSegments[0] - 1
		if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
			selectedUnit := g.Units.Units[unitIdx]

			unitLabelY := cfg.PickerCenterY - (cfg.RadiusInner / 2)
			DrawCenteredText(
				screen,
				fontFace,
				"Unit:",
				centerX,
				unitLabelY,
				400, // Увеличиваем ширину для длинных названий
				4,
				color.White,
			)

			unitNameY := unitLabelY + 35
			DrawCenteredText(
				screen,
				fontFace,
				selectedUnit.Name,
				centerX,
				unitNameY,
				400, // Увеличиваем ширину для длинных названий
				4,
				color.White,
			)
		}
	}

	if layerIndex == g.ActiveLayer && g.ActiveLayer >= 2 {
		unitIdx := g.SelectedSegments[0] - 1
		if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
			selectedUnit := g.Units.Units[unitIdx]
			if g.SelectedSegments[1] < len(selectedUnit.UnitNodes) {
				selectedNode := selectedUnit.UnitNodes[g.SelectedSegments[1]]

				unitNodeLabelY := cfg.PickerCenterY + (cfg.RadiusInner / 4)
				DrawCenteredText(
					screen,
					fontFace,
					"UnitNode:",
					centerX,
					unitNodeLabelY,
					400, // Увеличиваем ширину для длинных названий
					4,
					color.White,
				)

				unitNodeNameY := unitNodeLabelY + 35
				nodeName := selectedNode.TopicName
				if nodeName == "" {
					nodeName = selectedNode.UUID
				}
				DrawCenteredText(
					screen,
					fontFace,
					nodeName,
					centerX,
					unitNodeNameY,
					400, // Увеличиваем ширину для длинных названий
					4,
					color.White,
				)
			}
		}
	}

	if len(items[g.SelectedSegments[layerIndex]]) >= 2 {
		valueText := items[g.SelectedSegments[layerIndex]][1]

		if layerIndex == 2 {
			labelText := "Текст команды:"
			if pretty, ok := tryPrettyJSON(valueText); ok {
				valueText = pretty
				labelText = "JSON команды:"
			}

			labelY := centerY - fontSize/2
			labelWidth := text.BoundString(fontFace, labelText).Dx()
			labelX := valueColumnCenterX - labelWidth/2
			text.Draw(
				screen,
				labelText,
				fontFace,
				labelX,
				labelY,
				color.White,
			)

			valueTextY := labelY + fontSize + 10
			maxWidth := int(cfg.ScreenWidth / 5)
			valueTextX := valueColumnCenterX - maxWidth/2
			DrawLeftAlignedText(
				screen,
				fontFace,
				valueText,
				valueTextX,
				valueTextY,
				maxWidth,
				4,
				color.White,
			)
		} else {
			DrawCenteredText(
				screen,
				fontFace,
				valueText,
				centerX,
				centerOption+optionExternalLen+20,
				800,
				4,
				color.White,
			)
		}
	}

	if layerIndex == 2 {
		hotkeyLabel := "Горячие клавиши:"
		hotkeyValue := "Не установлены"

		if len(items[g.SelectedSegments[layerIndex]]) >= 3 {
			rawHotkey := strings.TrimSpace(items[g.SelectedSegments[layerIndex]][2])
			hotkeyValue = hotkeys.FormatHotkeyFromString(rawHotkey)
		}

		hotkeyColumnCenterX := cfg.ScreenWidth - valueColumnCenterX

		labelY := centerY - fontSize/2
		labelWidth := text.BoundString(fontFace, hotkeyLabel).Dx()
		labelX := hotkeyColumnCenterX - labelWidth/2
		text.Draw(
			screen,
			hotkeyLabel,
			fontFace,
			labelX,
			labelY,
			color.White,
		)

		hotkeyTextY := labelY + fontSize + 10
		maxWidth := int(cfg.ScreenWidth / 5)
		hotkeyTextX := hotkeyColumnCenterX - maxWidth/2

		displayText := hotkeyValue

		DrawLeftAlignedText(
			screen,
			fontFace,
			displayText,
			hotkeyTextX,
			hotkeyTextY,
			maxWidth,
			4,
			color.White,
		)
	}

	if layerIndex >= 0 {
		logLabelText := "Последние логи:"

		logColumnCenterX := cfg.ScreenWidth - valueColumnCenterX
		logLabelWidth := text.BoundString(fontFace, logLabelText).Dx()
		logLabelX := logColumnCenterX - logLabelWidth/2
		labelY := cfg.ScreenHeight * 1 / 2
		text.Draw(
			screen,
			logLabelText,
			fontFace,
			logLabelX,
			labelY,
			color.White,
		)

		logTextY := labelY + fontSize + 10
		logMaxWidth := int(cfg.ScreenWidth / 5)
		logTextX := logColumnCenterX - logMaxWidth/2

		logEntries := g.readLogEntries()
		logText := strings.Join(logEntries, "\n")

		logFontSize := 18
		logFontFace := LoadFont(float64(logFontSize))
		DrawLeftAlignedText(
			screen,
			logFontFace,
			logText,
			logTextX,
			logTextY,
			logMaxWidth,
			2,                              // Меньший межстрочный интервал
			color.RGBA{200, 200, 200, 255}, // Серый цвет для логов
		)

		unitIdx := g.SelectedSegments[0] - 1 // 0‑й сегмент — дефолтный
		if unitIdx >= 0 && unitIdx < len(g.Units.Units) && (layerIndex == 1 || layerIndex == 2) {
			selectedUnit := g.Units.Units[unitIdx]
			selectedNodeIdx := g.SelectedSegments[1]
			if selectedNodeIdx < len(selectedUnit.UnitNodes) {
				selectedNode := selectedUnit.UnitNodes[selectedNodeIdx]

				if g.lastNodeUnitIdx != unitIdx || g.lastNodeUnitNodeIdx != selectedNodeIdx || g.lastNodeInfoJSON == "" {
					nodeJSON, err := json.MarshalIndent(selectedNode, "", "    ")
					if err != nil {
						return
					}
					g.lastNodeInfoJSON = string(nodeJSON)
					g.lastNodeUnitIdx = unitIdx
					g.lastNodeUnitNodeIdx = selectedNodeIdx
				}

				labelText := "UnitNode Состояние:"

				labelWidth := text.BoundString(fontFace, labelText).Dx()
				labelX := valueColumnCenterX - labelWidth/2
				text.Draw(
					screen,
					labelText,
					fontFace,
					labelX,
					labelY,
					color.White,
				)

				valueTextY := labelY + fontSize + 10
				maxWidth := int(cfg.ScreenWidth / 5)
				valueTextX := valueColumnCenterX - maxWidth/2

				smallFontSize := 20
				smallFontFace := LoadFont(float64(smallFontSize))
				DrawLeftAlignedText(
					screen,
					smallFontFace,
					g.lastNodeInfoJSON,
					valueTextX,
					valueTextY,
					maxWidth,
					2, // Меньший межстрочный интервал
					color.White,
				)
			}
		}
	}

	switch {
	case g.ActiveLayer == 0 && layerIndex == 0:
		seg := g.SelectedSegments[0]
		var hintText string
		if seg == 0 {
			hintText = "ЛКМ: обновить список юнитов"
		} else if seg > 0 && seg <= len(g.Units.Units) {
			unit := g.Units.Units[seg-1]
			hintText = "ЛКМ: выбрать Unit | SPACE: открыть \"" + strings.TrimSpace(unit.Name) + "\" в браузере"
		}

		if hintText != "" {
			hintWidth := text.BoundString(fontFace, hintText).Dx()
			hintX := cfg.ScreenWidth/2 - hintWidth/2
			hintY := cfg.ScreenHeight - fontSize*2
			if hintY < 0 {
				hintY = fontSize
			}
			text.Draw(
				screen,
				hintText,
				fontFace,
				hintX,
				hintY,
				color.RGBA{200, 200, 200, 255},
			)
		}

	case g.ActiveLayer == 1 && layerIndex == 1:
		unitIdx := g.SelectedSegments[0] - 1
		nodeIdx := g.SelectedSegments[1]
		if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
			selectedUnit := g.Units.Units[unitIdx]
			if nodeIdx >= 0 && nodeIdx < len(selectedUnit.UnitNodes) {
				selectedNode := selectedUnit.UnitNodes[nodeIdx]

				entityName := strings.TrimSpace(selectedNode.TopicName)
				if entityName == "" {
					entityName = selectedNode.UUID
				}

				hintText := "ЛКМ: выбрать UnitNode | ПКМ: назад | SPACE: открыть \"" + entityName + "\" в браузере"

				hintWidth := text.BoundString(fontFace, hintText).Dx()
				hintX := cfg.ScreenWidth/2 - hintWidth/2
				hintY := cfg.ScreenHeight - fontSize*2
				if hintY < 0 {
					hintY = fontSize
				}
				text.Draw(
					screen,
					hintText,
					fontFace,
					hintX,
					hintY,
					color.RGBA{200, 200, 200, 255},
				)
			}
		}

	case g.ActiveLayer == 2 && layerIndex == 2:
		unitIdx := g.SelectedSegments[0] - 1
		selectedNodeIdx := g.SelectedSegments[1]
		if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
			selectedUnit := g.Units.Units[unitIdx]
			if selectedNodeIdx < len(selectedUnit.UnitNodes) {
				selectedNode := selectedUnit.UnitNodes[selectedNodeIdx]
				stateData := g.StateData[selectedNode.UUID]

				var hintText string
				if g.SelectedSegments[2] == 0 {
					hintText = "ЛКМ: создать новую команду | ПКМ: назад"
				} else if g.SelectedSegments[2] > 0 && g.SelectedSegments[2]-1 < len(stateData) {
					hintText = "ЛКМ: отправить команду | ПКМ: назад | DELETE: удалить команду | SPACE: установить хоткей | CTRL+SPACE: сбросить хоткей"
				}

				if hintText != "" {
					hintWidth := text.BoundString(fontFace, hintText).Dx()
					hintX := cfg.ScreenWidth/2 - hintWidth/2
					hintY := cfg.ScreenHeight - fontSize*2
					if hintY < 0 {
						hintY = fontSize
					}
					text.Draw(
						screen,
						hintText,
						fontFace,
						hintX,
						hintY,
						color.RGBA{200, 200, 200, 255},
					)
				}
			}
		}
	}
}

func (g *Game) drawMQTTStatus(screen *ebiten.Image) {
	if g.MQTTStatus == "" {
		return
	}

	cfg := config.GetConfig()

	fontFace := LoadFont(16)
	x := 20
	y := 40
	maxWidth := int(cfg.ScreenWidth / 3)

	DrawLeftAlignedText(
		screen,
		fontFace,
		g.MQTTStatus,
		x,
		y,
		maxWidth,
		2,
		color.RGBA{200, 200, 200, 255},
	)
}

func (g *Game) drawSpinner(screen *ebiten.Image) {
	if !g.spinnerActive || g.spinnerImage == nil {
		return
	}

	cfg := config.GetConfig()

	w, h := g.spinnerImage.Size()

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	op.GeoM.Rotate(g.spinnerAngle)
	op.GeoM.Translate(float64(cfg.PickerCenterX), float64(cfg.PickerCenterY))

	screen.DrawImage(g.spinnerImage, op)
}

func (g *Game) drawTextInputMessages(screen *ebiten.Image) {
	cfg := config.GetConfig()

	fontFace := LoadFont(24)
	fontBigFace := LoadFont(32)
	centerX := cfg.ScreenWidth / 2
	centerY := cfg.ScreenHeight / 2

	targetText := "Введите название команды"
	if !g.IsFirstWrite {
		targetText = "Введите тело команды: текст или JSON"
	}

	DrawCenteredText(
		screen,
		fontBigFace,
		targetText,
		centerX,
		centerY/3,
		300,
		4,
		color.White,
	)

	DrawCenteredText(
		screen,
		fontFace,
		"Введите текст или используйте <CTRL + v>",
		centerX,
		centerY/2,
		300,
		4,
		color.White,
	)

	DrawCenteredText(
		screen,
		fontFace,
		g.getTextInputWithCursor(),
		centerX,
		centerY,
		800,
		4,
		color.White,
	)

	hintText := "ENTER: сохранить | ESC: отменить"
	DrawCenteredText(
		screen,
		fontFace,
		hintText,
		centerX,
		centerY+100,
		600,
		4,
		color.RGBA{200, 200, 200, 255},
	)
}

func (g *Game) getTextInputWithCursor() string {
	const blinkPeriod = 60
	const halfPeriod = blinkPeriod / 2

	text := g.TextInput

	if blinkPeriod > 0 && (g.CursorTick%blinkPeriod) < halfPeriod {
		text += "|"
	}

	return text
}

func (g *Game) drawHotkeyInputMessages(screen *ebiten.Image) {
	cfg := config.GetConfig()

	fontFace := LoadFont(24)
	fontBigFace := LoadFont(32)
	centerX := cfg.ScreenWidth / 2
	centerY := cfg.ScreenHeight / 2

	optionName := g.HotkeyInputTargetOptionName
	if optionName == "" {
		optionName = "option"
	}

	targetText := "Горячие клавиши для команды: " + optionName

	DrawCenteredText(
		screen,
		fontBigFace,
		targetText,
		centerX,
		centerY/3,
		600,
		4,
		color.White,
	)

	DrawCenteredText(
		screen,
		fontFace,
		"Нажмите сочетание клавиш",
		centerX,
		centerY/2,
		400,
		4,
		color.White,
	)

	hotkeyDisplay := g.HotkeyInputCurrent
	if hotkeyDisplay == "" {
		hotkeyDisplay = "Клавиши ещё не нажаты"
	} else {
		hotkeyDisplay = hotkeys.FormatHotkeyFromString(hotkeyDisplay)
	}

	DrawCenteredText(
		screen,
		fontBigFace,
		hotkeyDisplay,
		centerX,
		centerY,
		600,
		4,
		color.RGBA{100, 200, 255, 255}, // Голубой цвет для текущей комбинации
	)

	hintText := "ENTER: сохранить | ESC: отменить | BACKSPACE/DELETE: Сбросить"
	DrawCenteredText(
		screen,
		fontFace,
		hintText,
		centerX,
		centerY+100,
		800,
		4,
		color.RGBA{200, 200, 200, 255},
	)
}
