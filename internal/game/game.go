package game

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"math"

	"github.com/atotto/clipboard"
	"github.com/hajimehoshi/ebiten/v2"

	"picker/internal/config"
	"picker/internal/graphics"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

type InputMode int

const (
	ModeGame InputMode = iota
	ModeTextInput
)

type Game struct {
	PepeClient       *pepeunit.PepeunitClient
	Units            UnitsByNodesResponse
	StateData        map[string][][]string
	KeyDownMap       map[ebiten.Key]bool // Состояние кнопок
	CursorTick       int                 // Счётчик для мигания курсора при вводе текста
	BackspaceFrames  int                 // Счётчик кадров удержания Backspace для автоповтора
	SelectedSegments []int               // Хранение текущего выбора для каждого слоя
	ActiveLayer      int                 // Индекс текущего слоя
	InputMode        InputMode
	TextInput        string
	OnTextInputDone  func(string)
	IsFirstWrite     bool
	// Кэш JSON‑представления выбранного UnitNode для уменьшения аллокаций в отрисовке.
	lastNodeInfoJSON    string
	lastNodeUnitIdx     int
	lastNodeUnitNodeIdx int
}

func (g *Game) GetState() map[string][][]string {
	// return a copy to avoid external mutation
	copyState := make(map[string][][]string)
	for uuid, options := range g.StateData {
		dup := make([][]string, len(options))
		for i, pair := range options {
			dup[i] = append([]string{}, pair...)
		}
		copyState[uuid] = dup
	}
	return copyState
}

func (g *Game) saveStateRemote() error {
	if g.PepeClient == nil || g.PepeClient.GetRESTClient() == nil {
		return nil
	}
	ctx := context.Background()
	payload, err := json.Marshal(g.StateData)
	if err != nil {
		return err
	}
	return g.PepeClient.SetStateStorage(ctx, string(payload))
}

func (g *Game) AddOption(unitNodeUUID, optionName, optionValue string) error {
	if _, ok := g.StateData[unitNodeUUID]; !ok {
		g.StateData[unitNodeUUID] = [][]string{}
	}
	// upsert
	for i, pair := range g.StateData[unitNodeUUID] {
		if len(pair) > 0 && pair[0] == optionName {
			// Обновляем значение, сохраняя при этом возможный хоткей (третье поле)
			if len(pair) == 1 {
				g.StateData[unitNodeUUID][i] = append(pair, optionValue)
			} else {
				g.StateData[unitNodeUUID][i][1] = optionValue
			}
			return g.saveStateRemote()
		}
	}
	g.StateData[unitNodeUUID] = append(g.StateData[unitNodeUUID], []string{optionName, optionValue})
	return g.saveStateRemote()
}

func (g *Game) RemoveOption(unitNodeUUID, optionName string) error {
	items, ok := g.StateData[unitNodeUUID]
	if !ok {
		return nil
	}
	filtered := make([][]string, 0, len(items))
	for _, pair := range items {
		if pair[0] != optionName {
			filtered = append(filtered, pair)
		}
	}
	g.StateData[unitNodeUUID] = filtered
	return g.saveStateRemote()
}

// SetOptionHotkey назначает хоткей для конкретной опции.
// Формат StateData: [name, value, hotkey] — третье поле опционально.
// Хоткей делаем глобально уникальным: перед назначением убираем его у других опций.
func (g *Game) SetOptionHotkey(unitNodeUUID, optionName, hotkey string) error {
	if hotkey == "" {
		return fmt.Errorf("hotkey cannot be empty")
	}

	// Снимаем этот хоткей со всех опций во всех нодах, чтобы он был глобально уникальным
	for nodeUUID, items := range g.StateData {
		for i, pair := range items {
			if len(pair) >= 3 && pair[2] == hotkey {
				g.StateData[nodeUUID][i][2] = ""
			}
		}
	}

	items, ok := g.StateData[unitNodeUUID]
	if !ok {
		return fmt.Errorf("unit node %s not found in state", unitNodeUUID)
	}

	for i, pair := range items {
		if len(pair) > 0 && pair[0] == optionName {
			switch len(pair) {
			case 1:
				// очень старый формат, только имя — расширяем до name, "", hotkey
				g.StateData[unitNodeUUID][i] = []string{pair[0], "", hotkey}
			case 2:
				// name, value — добавляем третье поле с хоткеем
				g.StateData[unitNodeUUID][i] = append(pair, hotkey)
			default:
				// уже есть третье поле — обновляем его
				g.StateData[unitNodeUUID][i][2] = hotkey
			}
			return g.saveStateRemote()
		}
	}

	return fmt.Errorf("option %s not found for unit node %s", optionName, unitNodeUUID)
}

// Метод для переключения в режим ввода текста
func (g *Game) StartTextInput(callback func(string)) {
	g.InputMode = ModeTextInput
	g.TextInput = ""
	g.OnTextInputDone = callback
	g.CursorTick = 0
	g.BackspaceFrames = 0
}

func (g *Game) AwaitTextInput(isFirstWrite bool) string {
	// Создаем канал для передачи текста
	resultChan := make(chan string)

	// Переключаем игру в режим ввода текста
	g.InputMode = ModeTextInput
	g.TextInput = ""
	g.IsFirstWrite = isFirstWrite
	g.CursorTick = 0
	g.BackspaceFrames = 0

	// Определяем колбэк для завершения ввода
	g.OnTextInputDone = func(input string) {
		resultChan <- input
		close(resultChan)
		g.InputMode = ModeGame
	}

	// Блокируем выполнение функции до получения результата
	return <-resultChan
}

func (g *Game) Update() error {
	switch g.InputMode {
	case ModeGame:
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
					// Внутри Game нет необходимости копировать всё состояние,
					// используем прямой доступ к данным, чтобы избежать лишних аллокаций.
					stateData := g.StateData[selectedNode.UUID]
					currentLayerLength = len(stateData) + 1
				}
			}
		}

		if currentLayerLength > 0 {
			segmentAngle := 2 * math.Pi / float64(currentLayerLength)
			g.SelectedSegments[g.ActiveLayer] = int(angle/segmentAngle) % currentLayerLength
		}

		g.handleKey(ebiten.KeyDelete, func() {
			if g.ActiveLayer == 2 {
				selectedUnitIdx := g.SelectedSegments[0]
				selectedNodeIdx := g.SelectedSegments[1]
				if selectedUnitIdx < len(g.Units.Units) {
					selectedUnit := g.Units.Units[selectedUnitIdx]
					if selectedNodeIdx < len(selectedUnit.UnitNodes) {
						selectedNode := selectedUnit.UnitNodes[selectedNodeIdx]
						stateData := g.StateData[selectedNode.UUID]
						if g.SelectedSegments[2] != 0 && g.SelectedSegments[2]-1 < len(stateData) {
							optionName := stateData[g.SelectedSegments[2]-1][0]
							// Используем uгRemoveOption для удаления опции
							err := g.RemoveOption(selectedNode.UUID, optionName)
							if err != nil {
								fmt.Println("Error removing option:", err)
							}
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
						stateData := g.StateData[selectedNode.UUID]
						if g.SelectedSegments[2] == 0 {
							fmt.Println("This is Add button")
							go func() {
								optionName := g.AwaitTextInput(true)
								optionContent := g.AwaitTextInput(false)
								g.AddOption(selectedNode.UUID, optionName, optionContent)
							}()
						} else {
							if stateData != nil {

								fmt.Println(stateData[g.SelectedSegments[2]-1])
								settings := g.PepeClient.GetSettings()
								topicName := settings.PU_DOMAIN + "/" + selectedNode.UUID + "/pepeunit"
								fmt.Println(topicName)
								if g.PepeClient != nil && g.PepeClient.GetMQTTClient() != nil {
									err := g.PepeClient.GetMQTTClient().Publish(topicName, stateData[g.SelectedSegments[2]-1][1])
									if err == nil {
										fmt.Println("Sendet")
									}
								}
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

		// Назначение хоткея для опции на третьем бублике:
		// при активном третьем слое и выбранной опции (кроме "Create New Option")
		// по Ctrl+Shift+<буква A-Z> записываем хоткей в состояние.
		if g.ActiveLayer == 2 {
			selectedUnitIdx := g.SelectedSegments[0]
			selectedNodeIdx := g.SelectedSegments[1]

			if selectedUnitIdx < len(g.Units.Units) {
				selectedUnit := g.Units.Units[selectedUnitIdx]
				if selectedNodeIdx < len(selectedUnit.UnitNodes) {
					selectedNode := selectedUnit.UnitNodes[selectedNodeIdx]
					stateData := g.StateData[selectedNode.UUID]

					// 0‑й сегмент — "Create New Option", реальные опции начинаются с индекса 1
					if g.SelectedSegments[2] > 0 && g.SelectedSegments[2]-1 < len(stateData) {
						optionName := stateData[g.SelectedSegments[2]-1][0]
						g.handleHotkeyAssignment(optionName, selectedNode.UUID)
					}
				}
			}
		}
	case ModeTextInput:
		// Обновляем счётчик мигания курсора
		g.CursorTick++

		for _, char := range ebiten.InputChars() {
			if char != '\n' && char != '\r' {
				g.TextInput += string(char)
			}
		}

		// Обработка Backspace с автоповтором при удержании
		if ebiten.IsKeyPressed(ebiten.KeyBackspace) {
			g.BackspaceFrames++

			const initialDelay = 15  // задержка перед началом автоповтора (~0.25с при 60 FPS)
			const repeatInterval = 3 // интервал автоповтора (~20 удалений в секунду)

			// Удаляем символ:
			// - сразу при первом нажатии
			// - затем через initialDelay кадров
			// - потом с периодом repeatInterval кадров
			if g.BackspaceFrames == 1 ||
				(g.BackspaceFrames > initialDelay && (g.BackspaceFrames-initialDelay)%repeatInterval == 0) {
				if len(g.TextInput) > 0 {
					g.TextInput = g.TextInput[:len(g.TextInput)-1]
				}
			}
		} else {
			// Клавишу отпустили — сбрасываем счётчик, чтобы не было «залипания»
			g.BackspaceFrames = 0
		}

		g.handleKeyCombination(ebiten.KeyV, ebiten.KeyControl, func() {
			clipboardText, err := clipboard.ReadAll()
			if err == nil {
				g.TextInput += clipboardText
			}
		})

		g.handleKey(ebiten.KeyEnter, func() {
			if g.OnTextInputDone != nil {
				g.OnTextInputDone(g.TextInput)
			}
			g.InputMode = ModeGame

		})

	}

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

func (g *Game) handleKeyCombination(key ebiten.Key, modifier ebiten.Key, action func()) {
	if ebiten.IsKeyPressed(key) && ebiten.IsKeyPressed(modifier) {
		if !g.KeyDownMap[key] {
			g.KeyDownMap[key] = true
			action()
		}
	} else {
		g.KeyDownMap[key] = false
	}
}

// handleHotkeyAssignment отслеживает нажатия Ctrl+Shift+буква (A‑Z)
// и при первом нажатии назначает хоткей для указанной опции.
func (g *Game) handleHotkeyAssignment(optionName, unitNodeUUID string) {
	for k := ebiten.KeyA; k <= ebiten.KeyZ; k++ {
		if ebiten.IsKeyPressed(k) && ebiten.IsKeyPressed(ebiten.KeyControl) && ebiten.IsKeyPressed(ebiten.KeyShift) {
			if !g.KeyDownMap[k] {
				g.KeyDownMap[k] = true

				// Преобразуем код клавиши в букву A‑Z
				offset := int(k - ebiten.KeyA)
				if offset >= 0 && offset < 26 {
					hotkeyChar := string(rune('A' + offset))
					if err := g.SetOptionHotkey(unitNodeUUID, optionName, hotkeyChar); err != nil {
						fmt.Println("failed to set hotkey:", err)
					} else {
						fmt.Printf("assigned hotkey Ctrl+Shift+%s to option %s (%s)\n", hotkeyChar, optionName, unitNodeUUID)
					}
				}
			}
		} else {
			g.KeyDownMap[k] = false
		}
	}
}

func (g *Game) Draw(screen *ebiten.Image) {

	cfg := config.GetConfig()
	if cfg.BlurredBackground == nil {
		g.drawBlurLoadingMessage(screen)
		return
	}
	// Размытый фон уже закеширован как *ebiten.Image,
	// избегаем создания нового ebiten.Image каждый кадр.
	screen.DrawImage(cfg.BlurredBackground, nil)

	switch g.InputMode {
	case ModeGame:
		for layerIndex := 0; layerIndex <= g.ActiveLayer; layerIndex++ {
			var items [][]string

			switch layerIndex {
			case 0:
				for _, unit := range g.Units.Units {
					items = append(items, []string{unit.Name})
				}
			case 1:
				if g.SelectedSegments[0] < len(g.Units.Units) {
					for _, node := range g.Units.Units[g.SelectedSegments[0]].UnitNodes {
						items = append(items, []string{node.TopicName})
					}
				}
			case 2:
				if g.SelectedSegments[0] < len(g.Units.Units) {
					selectedUnit := g.Units.Units[g.SelectedSegments[0]]
					if g.SelectedSegments[1] < len(selectedUnit.UnitNodes) {
						selectedNode := selectedUnit.UnitNodes[g.SelectedSegments[1]]
						stateData := g.StateData[selectedNode.UUID]

						items = append(items, []string{"Create New Option"})

						items = append(items, stateData...)
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
					angleStart+0.01-0.0015*float64(layerIndex),
					angleEnd-0.01+0.0015*float64(layerIndex),
					clr,
				)
			}
			g.drawGameModeMessages(screen, layerIndex, items)
		}
	case ModeTextInput:
		g.drawTextInputMessages(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	cfg := config.GetConfig()
	return cfg.ScreenWidth, cfg.ScreenHeight
}
