package game

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"image/color"
	"log"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"picker/internal/config"
	"picker/internal/hotkeys"
)

//go:embed fonts/cornerita_black.ttf
var fontData []byte

var (
	baseFont  *opentype.Font
	fontCache = make(map[float64]font.Face)
	fontMu    sync.Mutex
)

// LoadFont загружает и кэширует шрифт нужного размера.
// Парсинг TTF и создание Face — тяжёлая операция, поэтому мы делаем её один раз
// для каждого размера и переиспользуем результат между кадрами.
func LoadFont(size float64) font.Face {
	fontMu.Lock()
	defer fontMu.Unlock()

	if face, ok := fontCache[size]; ok {
		return face
	}

	if baseFont == nil {
		tt, err := opentype.Parse(fontData)
		if err != nil {
			log.Fatalf("failed to parse font: %v", err)
		}
		baseFont = tt
	}

	face, err := opentype.NewFace(baseFont, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatalf("failed to create font face: %v", err)
	}

	fontCache[size] = face
	return face
}

// DrawCenteredText отрисовывает большой текст с центрированием
func DrawCenteredText(screen *ebiten.Image, face font.Face, textContent string, x, y, maxWidth, lineSpacing int, clr color.Color) {
	lines := wrapText(face, textContent, maxWidth)
	lineHeight := text.BoundString(face, "A").Dy() + lineSpacing
	totalHeight := len(lines) * lineHeight
	startY := y - totalHeight/2

	for i, line := range lines {
		lineWidth := text.BoundString(face, line).Dx()
		startX := x - lineWidth/2
		text.Draw(screen, line, face, startX, startY+(i*lineHeight), clr)
	}
}

// DrawLeftAlignedText отрисовывает текст с переносами строк и выравниванием по левому краю
func DrawLeftAlignedText(screen *ebiten.Image, face font.Face, textContent string, x, y, maxWidth, lineSpacing int, clr color.Color) {
	lineHeight := text.BoundString(face, "A").Dy() + lineSpacing
	currentY := y

	paragraphs := strings.Split(textContent, "\n")
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			currentY += lineHeight
			continue
		}

		lines := wrapText(face, p, maxWidth)
		for _, line := range lines {
			text.Draw(screen, line, face, x, currentY, clr)
			currentY += lineHeight
		}
	}
}

// wrapText разбивает текст на строки, которые помещаются в указанную ширину.
// Работает и для обычного текста с пробелами, и для длинных "слов" без пробелов (например, токенов/JWT).
func wrapText(face font.Face, textContent string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{textContent}
	}

	runes := []rune(textContent)
	var lines []string
	var current []rune
	lastSpace := -1

	for i, r := range runes {
		current = append(current, r)
		if r == ' ' || r == '\t' {
			lastSpace = len(current) - 1
		}

		if text.BoundString(face, string(current)).Dx() > maxWidth {
			breakPos := len(current) - 1
			if lastSpace >= 0 {
				breakPos = lastSpace
			}

			lineRunes := current[:breakPos]
			line := strings.TrimSpace(string(lineRunes))
			if line != "" {
				lines = append(lines, line)
			}

			// Оставшуюся часть текущей строки переносим на следующую итерацию
			if breakPos < len(current) {
				current = current[breakPos:]
			} else {
				current = []rune{}
			}

			// Пересчитываем lastSpace для оставшихся рун
			lastSpace = -1
			for j, rr := range current {
				if rr == ' ' || rr == '\t' {
					lastSpace = j
				}
			}
		}

		// Если это последний символ — добавляем текущую строку
		if i == len(runes)-1 {
			line := strings.TrimSpace(string(current))
			if line != "" {
				lines = append(lines, line)
			}
		}
	}

	if len(lines) == 0 {
		return []string{""}
	}

	return lines
}

// tryPrettyJSON пытается распарсить строку как JSON и вернуть форматированный вывод
func tryPrettyJSON(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	if !json.Valid([]byte(raw)) {
		return "", false
	}

	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(raw), "", "    "); err != nil {
		return "", false
	}

	return buf.String(), true
}

// drawBlurLoadingMessage выводит сообщение о загрузке размытого фона
func (g *Game) drawBlurLoadingMessage(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "Загрузка размытого фона...")
}

// drawGameModeMessages отвечает за отрисовку подписей сегментов в игровом режиме
func (g *Game) drawGameModeMessages(screen *ebiten.Image, layerIndex int, items [][]string) {
	if len(items) == 0 {
		return
	}

	// Показываем подписи только для активного слоя
	if layerIndex != g.ActiveLayer {
		return
	}

	if g.SelectedSegments[layerIndex] < 0 || len(items) <= g.SelectedSegments[layerIndex] {
		return
	}

	cfg := config.GetConfig()

	fontSize := 24
	fontFace := LoadFont(float64(fontSize))

	// Отдельный больший шрифт для надписи выбираемого сегмента
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

	// Основной текст (название элемента) - выводится наверху бублика
	// для выбранного сегмента с увеличенным лимитом символов (минимум 50)
	selectedSegmentText := items[g.SelectedSegments[layerIndex]][0]
	maxWidth := cfg.RadiusInner * 2 // Увеличиваем ширину для размещения большего количества символов
	if maxWidth < 50*12 {           // 50 символов примерно по 12 пикселей каждый
		maxWidth = 50 * 12
	}

	// Позиция наверху самого внешнего бублика (активного слоя)
	activeLayerOuterRadius := cfg.RadiusInner + layerIndex*60 + cfg.ThickSegment
	segmentLabelY := cfg.PickerCenterY - activeLayerOuterRadius - 33 // 33 пикселя выше внешнего края активного слоя (учитывая больший шрифт)

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

	// Отображение выбранного Unit в центре бублика (для слоев >= 1)
	if layerIndex == g.ActiveLayer && g.ActiveLayer >= 1 {
		unitIdx := g.SelectedSegments[0] - 1
		if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
			selectedUnit := g.Units.Units[unitIdx]

			// "Unit:" в середине между центром бублика и внутренним краем первого сегмента (верхняя часть)
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

			// Название Unit ниже метки
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

	// Отображение выбранного UnitNode в центре бублика (для слоев >= 2)
	if layerIndex == g.ActiveLayer && g.ActiveLayer >= 2 {
		unitIdx := g.SelectedSegments[0] - 1
		if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
			selectedUnit := g.Units.Units[unitIdx]
			if g.SelectedSegments[1] < len(selectedUnit.UnitNodes) {
				selectedNode := selectedUnit.UnitNodes[g.SelectedSegments[1]]

				// "UnitNode:" в середине между центром бублика и внутренним краем первого сегмента (нижняя часть)
				unitNodeLabelY := cfg.PickerCenterY + (cfg.RadiusInner / 2)
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

				// Название UnitNode ниже метки
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

	// Дополнительный текст (значение опции), если он есть
	if len(items[g.SelectedSegments[layerIndex]]) >= 2 {
		valueText := items[g.SelectedSegments[layerIndex]][1]

		// Для третьего бублика (options) отображаем значение слева от бублика
		if layerIndex == 2 {
			// Пытаемся отрендерить значение как JSON
			labelText := "Текст команды:"
			if pretty, ok := tryPrettyJSON(valueText); ok {
				valueText = pretty
				labelText = "JSON команды:"
			}

			// Надпись слева, по высоте примерно на уровне названия сегмента
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

			// Текст значения на 20-40 пикселей ниже "Value:" и ограничен по ширине четвертью экрана
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
			// Для остальных слоёв (если появятся значения) сохраняем старое поведение
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

	// Отображение текущего хоткея для опции на третьем бублике — в правой колонке,
	// по аналогии с текстом под "Value:".
	if layerIndex == 2 {
		hotkeyLabel := "Горячие клавиши:"
		hotkeyValue := "Не установлены"

		if len(items[g.SelectedSegments[layerIndex]]) >= 3 {
			rawHotkey := strings.TrimSpace(items[g.SelectedSegments[layerIndex]][2])
			hotkeyValue = hotkeys.FormatHotkeyFromString(rawHotkey)
		}

		// Центр правой колонки симметрично левой
		hotkeyColumnCenterX := cfg.ScreenWidth - valueColumnCenterX

		// Надпись "Hotkey:" справа, по высоте на уровне названия сегмента
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

		// Текущая комбинация под надписью, ограниченная по ширине так же, как и слева
		hotkeyTextY := labelY + fontSize + 10
		maxWidth := int(cfg.ScreenWidth / 5)
		hotkeyTextX := hotkeyColumnCenterX - maxWidth/2

		// Показываем только значение хоткея
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

	// На втором и третьем бубликах дополнительно показываем JSON‑информацию о UnitNode слева
	if layerIndex == 1 || layerIndex == 2 {
		unitIdx := g.SelectedSegments[0] - 1 // 0‑й сегмент — дефолтный
		if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
			selectedUnit := g.Units.Units[unitIdx]
			selectedNodeIdx := g.SelectedSegments[1]
			if selectedNodeIdx < len(selectedUnit.UnitNodes) {
				selectedNode := selectedUnit.UnitNodes[selectedNodeIdx]

				// Кэшируем JSON‑представление UnitNode, чтобы не сериализовать каждый кадр.
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

				// Надпись слева, зеркально "Текст команды:" в нижней половине экрана
				// Рассчитываем centerY как для третьего слоя
				optionsCenterY := centerOption - optionExternalLen + fontSize
				// Зеркальное отражение относительно середины экрана
				labelY := cfg.ScreenHeight - optionsCenterY + fontSize/2
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

				// Форматированный JSON ниже подписи, ограничен по ширине колонки
				valueTextY := labelY + fontSize + 10
				maxWidth := int(cfg.ScreenWidth / 5)
				valueTextX := valueColumnCenterX - maxWidth/2

				// Используем меньший шрифт для JSON чтобы уместить на экране
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

	// Подсказки под бубликами с информацией о доступных действиях
	switch {
	// Первый бублик
	case g.ActiveLayer == 0 && layerIndex == 0:
		seg := g.SelectedSegments[0]
		var hintText string
		if seg == 0 {
			// 0‑й сегмент — "Обновить список юнитов"
			hintText = "ЛКМ: обновить список юнитов"
		} else if seg > 0 && seg <= len(g.Units.Units) {
			// Выбран конкретный Unit
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

	// Второй бублик
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

	// Третий бублик
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
					// "Create New Option"
					hintText = "ЛКМ: создать новую команду | ПКМ: назад"
				} else if g.SelectedSegments[2] > 0 && g.SelectedSegments[2]-1 < len(stateData) {
					// Выбрана существующая опция
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

// drawMQTTStatus выводит текстовый статус MQTT‑соединения в левом верхнем углу.
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

// drawSpinner рисует спинер в центре бублика, если он активен.
func (g *Game) drawSpinner(screen *ebiten.Image) {
	if !g.spinnerActive || g.spinnerImage == nil {
		return
	}

	cfg := config.GetConfig()

	w, h := g.spinnerImage.Size()

	op := &ebiten.DrawImageOptions{}
	// Центрируем изображение относительно (0,0)
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	// Поворот вокруг центра
	op.GeoM.Rotate(g.spinnerAngle)
	// Перенос в центр бублика
	op.GeoM.Translate(float64(cfg.PickerCenterX), float64(cfg.PickerCenterY))

	screen.DrawImage(g.spinnerImage, op)
}

// drawTextInputMessages выводит подсказки и введённый текст в режиме ввода
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

	// Подсказки для клавиш
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

// getTextInputWithCursor возвращает строку ввода с мигающим курсором
func (g *Game) getTextInputWithCursor() string {
	// Период мигания ~0.5 секунды при 60 тиках в секунду:
	// 30 кадров курсор виден, 30 кадров скрыт.
	const blinkPeriod = 60
	const halfPeriod = blinkPeriod / 2

	text := g.TextInput

	if blinkPeriod > 0 && (g.CursorTick%blinkPeriod) < halfPeriod {
		// Добавляем простой вертикальный курсор
		text += "|"
	}

	return text
}

// drawHotkeyInputMessages выводит подсказки и текущую комбинацию клавиш в режиме ввода хоткея
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

	// Отображаем текущую комбинацию клавиш
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

	// Подсказки
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
