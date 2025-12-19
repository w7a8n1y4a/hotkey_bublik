package game

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/hajimehoshi/ebiten/v2"

	"picker/internal/config"
	"picker/internal/graphics"
	"picker/internal/hotkeys"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

type InputMode int

const (
	ModeGame InputMode = iota
	ModeTextInput
	ModeHotkeyInput
)

// Структура для записи лога
type LogEntry struct {
	CreateDatetime string `json:"create_datetime"`
	Level          string `json:"level"`
	Text           string `json:"text"`
}

// Палитра базовых цветов Unit, близкая к Google Material Design.
// Используется для первого бублика: цвет сегмента определяется по индексу Unit через остаток от деления на 10.
var unitColors = []color.RGBA{
	{0xF4, 0x43, 0x36, 0xFF}, // 0 → красный      (#F44336)
	{0xE9, 0x1E, 0x63, 0xFF}, // 1 → розовый      (#E91E63)
	{0x9C, 0x27, 0xB0, 0xFF}, // 2 → фиолетовый   (#9C27B0)
	{0x3F, 0x51, 0xB5, 0xFF}, // 3 → индиго       (#3F51B5)
	{0x21, 0x96, 0xF3, 0xFF}, // 4 → синий        (#2196F3)
	{0x03, 0xA9, 0xF4, 0xFF}, // 5 → голубой      (#03A9F4)
	{0x00, 0x96, 0x88, 0xFF}, // 6 → бирюзовый    (#009688)
	{0x4C, 0xAF, 0x50, 0xFF}, // 7 → зелёный      (#4CAF50)
	{0xFF, 0x98, 0x00, 0xFF}, // 8 → оранжевый    (#FF9800)
	{0xFF, 0x57, 0x22, 0xFF}, // 9 → тёплый оранж (#FF5722)
}

// Тёмный дефолтный цвет сегмента для всех слоёв.
var defaultSegmentColor = color.RGBA{0x42, 0x42, 0x42, 0xFF} // #424242

// Цвет дефолтного сегмента "Обновить список юнитов" при наведении.
var refreshSegmentColor = color.RGBA{0x60, 0x7D, 0x8B, 0xFF} // #607D8B

type Game struct {
	PepeClient        *pepeunit.PepeunitClient
	Units             UnitsByNodesResponse
	StateData         map[string][][]string
	KeyDownMap        map[ebiten.Key]bool // Состояние кнопок
	CursorTick        int                 // Счётчик для мигания курсора при вводе текста
	BackspaceFrames   int                 // Счётчик кадров удержания Backspace для автоповтора
	SelectedSegments  []int               // Хранение текущего выбора для каждого слоя
	ActiveLayer       int                 // Индекс текущего слоя
	InputMode         InputMode
	TextInput         string
	OnTextInputDone   func(string)
	OnTextInputCancel func()
	IsFirstWrite      bool
	// Режим ввода хоткеев
	HotkeyInputTargetUnitNodeUUID string
	HotkeyInputTargetOptionName   string
	HotkeyInputCurrent            string
	OnHotkeyInputDone             func(string)
	OnHotkeyInputCancel           func()
	// Кэш JSON‑представления выбранного UnitNode для уменьшения аллокаций в отрисовке.
	lastNodeInfoJSON    string
	lastNodeUnitIdx     int
	lastNodeUnitNodeIdx int

	// Кэш последних логов для отображения
	lastLogEntries    []string
	lastLogUpdateTime time.Time

	// Спинер загрузки/отправки
	spinnerImage       *ebiten.Image
	spinnerActive      bool
	spinnerAngle       float64
	spinnerStart       time.Time
	spinnerLastUpdate  time.Time
	spinnerOpsInFlight int
	spinnerMinDuration time.Duration

	// Асинхронные операции
	refreshResultCh   chan refreshResult
	refreshInProgress bool
	mqttResultCh      chan mqttResult
	mqttInProgress    bool

	// MQTTStatus хранит человекочитаемый статус MQTT‑соединения / последней операции.
	MQTTStatus string
}

type refreshResult struct {
	data UnitsByNodesResponse
	err  error
}

type mqttResult struct {
	err error
}

// NewGame конструирует Game, подготавливая спинер и каналы для асинхронных операций.
func NewGame(client *pepeunit.PepeunitClient, data UnitsByNodesResponse, stateData map[string][][]string) (*Game, error) {
	cfg := config.GetConfig()

	// Размер спинера: примерно в 2 раза больше предыдущего варианта.
	// Можно тонко настроить коэффициент при необходимости.
	spinnerSize := 2 * (cfg.RadiusInner - 40)
	if spinnerSize < 10 {
		spinnerSize = 10
	}

	spinnerImg, err := loadSpinnerImage(spinnerSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load spinner image: %w", err)
	}

	mqttStatus := "MQTT: disabled"
	if client != nil && client.GetMQTTClient() != nil {
		mqttStatus = "MQTT: ready"
	}

	g := &Game{
		PepeClient:         client,
		Units:              data,
		StateData:          stateData,
		KeyDownMap:         make(map[ebiten.Key]bool),
		SelectedSegments:   make([]int, 3),
		ActiveLayer:        0,
		spinnerImage:       spinnerImg,
		spinnerMinDuration: 100 * time.Millisecond,
		refreshResultCh:    make(chan refreshResult, 1),
		mqttResultCh:       make(chan mqttResult, 1),
		MQTTStatus:         mqttStatus,
	}

	return g, nil
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

// FetchUnits загружает список Unit и UnitNode через REST‑клиент pepeunit.
// При отсутствии доступных юнитов возвращает пустой результат БЕЗ ошибки,
// чтобы приложение могло запускаться даже в таком состоянии.
func FetchUnits(client *pepeunit.PepeunitClient) (UnitsByNodesResponse, error) {
	if client == nil || client.GetRESTClient() == nil {
		return UnitsByNodesResponse{}, fmt.Errorf("REST client is not initialized")
	}
	if client.GetSchema() == nil {
		return UnitsByNodesResponse{}, fmt.Errorf("schema is not initialized")
	}

	// Находим URL output_units_nodes/pepeunit в schema.json
	outputTopics := client.GetSchema().GetOutputTopic()
	topicURLs, ok := outputTopics["output_units_nodes/pepeunit"]
	if !ok || len(topicURLs) == 0 {
		// Нет топиков — считаем, что просто нет доступных юнитов.
		return UnitsByNodesResponse{}, nil
	}
	topicURL := topicURLs[0]

	// Валидация URL (на всякий случай). Некорректный URL трактуем
	// как отсутствие доступных юнитов, а не как фатальную ошибку.
	if _, err := url.Parse(topicURL); err != nil {
		return UnitsByNodesResponse{}, nil
	}

	ctx := context.Background()

	// 1. Получаем UnitNodes по output topic
	rawNodes, err := client.GetRESTClient().GetInputByOutput(ctx, topicURL, 100, 0)
	if err != nil {
		return UnitsByNodesResponse{}, err
	}
	nodesBytes, err := json.Marshal(rawNodes)
	if err != nil {
		return UnitsByNodesResponse{}, err
	}

	var unitNodesResp UnitNodesResponse
	if err := json.Unmarshal(nodesBytes, &unitNodesResp); err != nil {
		return UnitsByNodesResponse{}, err
	}

	if unitNodesResp.Count == 0 || len(unitNodesResp.UnitNodes) == 0 {
		// Нет связей — возвращаем пустой результат без ошибки.
		return UnitsByNodesResponse{}, nil
	}

	unitNodeUUIDs := make([]string, 0, len(unitNodesResp.UnitNodes))
	for _, item := range unitNodesResp.UnitNodes {
		unitNodeUUIDs = append(unitNodeUUIDs, item.UUID)
	}

	// 2. Получаем Units по UUID узлов
	rawUnits, err := client.GetRESTClient().GetUnitsByNodes(ctx, unitNodeUUIDs, 100, 0)
	if err != nil {
		return UnitsByNodesResponse{}, err
	}
	unitsBytes, err := json.Marshal(rawUnits)
	if err != nil {
		return UnitsByNodesResponse{}, err
	}

	var unitsResp UnitsByNodesResponse
	if err := json.Unmarshal(unitsBytes, &unitsResp); err != nil {
		return UnitsByNodesResponse{}, err
	}

	return unitsResp, nil
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
// Если hotkey пустая строка, хоткей удаляется.
func (g *Game) SetOptionHotkey(unitNodeUUID, optionName, hotkey string) error {
	// Пустая строка означает удаление хоткея
	if hotkey == "" {
		// Просто удаляем хоткей, не проверяя уникальность
		items, ok := g.StateData[unitNodeUUID]
		if !ok {
			return fmt.Errorf("unit node %s not found in state", unitNodeUUID)
		}

		for i, pair := range items {
			if len(pair) > 0 && pair[0] == optionName {
				if len(pair) >= 3 {
					g.StateData[unitNodeUUID][i][2] = ""
				}
				return g.saveStateRemote()
			}
		}
		return fmt.Errorf("option %s not found for unit node %s", optionName, unitNodeUUID)
	}

	// Валидация хоткея
	if err := hotkeys.ValidateHotkey(hotkey); err != nil {
		return fmt.Errorf("invalid hotkey: %w", err)
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
	g.OnTextInputCancel = nil
	g.CursorTick = 0
	g.BackspaceFrames = 0
}

type textInputResult struct {
	text      string
	cancelled bool
}

func (g *Game) AwaitTextInput(isFirstWrite bool) (string, bool) {
	// Создаем канал для передачи текста
	resultChan := make(chan textInputResult, 1)

	// Переключаем игру в режим ввода текста
	g.InputMode = ModeTextInput
	g.TextInput = ""
	g.IsFirstWrite = isFirstWrite
	g.CursorTick = 0
	g.BackspaceFrames = 0

	// Определяем колбэк для завершения ввода
	finish := func(res textInputResult) {
		// На всякий случай: после завершения сбрасываем колбэки,
		// чтобы повторные нажатия не пытались писать в уже закрытый канал.
		g.OnTextInputDone = nil
		g.OnTextInputCancel = nil
		g.InputMode = ModeGame
		resultChan <- res
		close(resultChan)
	}

	g.OnTextInputDone = func(input string) { finish(textInputResult{text: input, cancelled: false}) }
	g.OnTextInputCancel = func() { finish(textInputResult{text: "", cancelled: true}) }

	// Блокируем выполнение функции до получения результата
	res := <-resultChan
	return res.text, res.cancelled
}

func (g *Game) Update() error {
	g.updateSpinner()

	switch g.InputMode {
	case ModeGame:
		// ESC закрывает игру только по "первому нажатию" (edge-trigger),
		// иначе после отмены ввода в ModeTextInput (где ESC = назад) клавиша
		// остаётся зажатой и приводила к мгновенному выходу на следующем тике.
		if ebiten.IsKeyPressed(ebiten.KeyEscape) {
			if !g.KeyDownMap[ebiten.KeyEscape] {
				g.KeyDownMap[ebiten.KeyEscape] = true
				return fmt.Errorf("game closed by user")
			}
		} else {
			g.KeyDownMap[ebiten.KeyEscape] = false
		}

		// Обрабатываем результаты асинхронных операций (refresh units, MQTT publish).
		select {
		case res := <-g.refreshResultCh:
			g.refreshInProgress = false
			if res.err != nil {
				fmt.Println("failed to refresh units:", res.err)
			} else {
				g.Units = res.data
				g.resetSelection()
			}
			g.finishSpinnerOp()
		default:
		}

		select {
		case res := <-g.mqttResultCh:
			g.mqttInProgress = false
			if res.err != nil {
				fmt.Println("failed to publish MQTT message:", res.err)
				g.MQTTStatus = "MQTT: error: " + res.err.Error()
			} else {
				g.MQTTStatus = "MQTT: last publish OK"
			}
			g.finishSpinnerOp()
		default:
		}

		cfg := config.GetConfig()
		mouseX, mouseY := ebiten.CursorPosition()
		dx, dy := mouseX-cfg.PickerCenterX, mouseY-cfg.PickerCenterY
		angle := math.Atan2(-float64(dy), -float64(dx)) + math.Pi

		var currentLayerLength int
		switch g.ActiveLayer {
		case 0: // Первый слой — список Units
			// +1 — дефолтный сегмент для обновления списка юнитов.
			currentLayerLength = len(g.Units.Units) + 1
		case 1: // Второй слой — UnitNodes выбранного Unit
			unitIdx := g.SelectedSegments[0] - 1 // 0‑й сегмент — дефолтный
			if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
				currentLayerLength = len(g.Units.Units[unitIdx].UnitNodes)
			}
		case 2: // Третий слой — данные из StateManager
			unitIdx := g.SelectedSegments[0] - 1
			if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
				selectedUnit := g.Units.Units[unitIdx]
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

		// Клавиша Space:
		// - на первом бублике открывает страницу unit в браузере;
		// - на втором бублике открывает страницу unit-node в браузере;
		// - на третьем бублике открывает режим ввода хоткея для выбранной опции.
		g.handleKey(ebiten.KeySpace, func() {
			switch g.ActiveLayer {
			case 0:
				// Первый бублик: открываем страницу Unit.
				settings := g.PepeClient.GetSettings()
				if g.PepeClient == nil {
					return
				}
				unitIdx := g.SelectedSegments[0] - 1 // 0‑й сегмент — "Обновить список юнитов"
				if unitIdx < 0 || unitIdx >= len(g.Units.Units) {
					return
				}
				selectedUnit := g.Units.Units[unitIdx]
				unitURL := fmt.Sprintf("%s://%s/unit/%s", settings.PU_HTTP_TYPE, settings.PU_DOMAIN, selectedUnit.UUID)

				go func(url string) {
					cmd := exec.Command("xdg-open", url)
					_ = cmd.Start()
				}(unitURL)

			case 1:
				// Второй бублик: открываем страницу UnitNode.
				settings := g.PepeClient.GetSettings()
				if g.PepeClient == nil {
					return
				}
				unitIdx := g.SelectedSegments[0] - 1
				selectedNodeIdx := g.SelectedSegments[1]

				if unitIdx < 0 || unitIdx >= len(g.Units.Units) {
					return
				}
				selectedUnit := g.Units.Units[unitIdx]
				if selectedNodeIdx < 0 || selectedNodeIdx >= len(selectedUnit.UnitNodes) {
					return
				}
				selectedNode := selectedUnit.UnitNodes[selectedNodeIdx]

				unitNodeURL := fmt.Sprintf("%s://%s/unit-node/%s", settings.PU_HTTP_TYPE, settings.PU_DOMAIN, selectedNode.UUID)

				go func(url string) {
					cmd := exec.Command("xdg-open", url)
					_ = cmd.Start()
				}(unitNodeURL)

			case 2:
				// Третий бублик: Space - установить хоткей, Ctrl+Space - сбросить хоткей
				unitIdx := g.SelectedSegments[0] - 1
				selectedNodeIdx := g.SelectedSegments[1]
				if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
					selectedUnit := g.Units.Units[unitIdx]
					if selectedNodeIdx < len(selectedUnit.UnitNodes) {
						selectedNode := selectedUnit.UnitNodes[selectedNodeIdx]
						stateData := g.StateData[selectedNode.UUID]
						if g.SelectedSegments[2] > 0 && g.SelectedSegments[2]-1 < len(stateData) {
							optionName := stateData[g.SelectedSegments[2]-1][0]

							// Если нажат Ctrl - сбрасываем хоткей, иначе - устанавливаем новый
							if ebiten.IsKeyPressed(ebiten.KeyControl) {
								// Сброс хоткея
								if err := g.SetOptionHotkey(selectedNode.UUID, optionName, ""); err != nil {
									fmt.Println("Error clearing hotkey:", err)
								}
							} else {
								// Установка хоткея
								go func() {
									hotkey, cancelled := g.AwaitHotkeyInput(selectedNode.UUID, optionName)
									if cancelled {
										return
									}
									if err := g.SetOptionHotkey(selectedNode.UUID, optionName, hotkey); err != nil {
										fmt.Println("Error setting hotkey:", err)
									}
								}()
							}
						}
					}
				}
			}
		})

		g.handleKey(ebiten.KeyDelete, func() {
			if g.ActiveLayer == 2 {
				unitIdx := g.SelectedSegments[0] - 1
				selectedNodeIdx := g.SelectedSegments[1]
				if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
					selectedUnit := g.Units.Units[unitIdx]
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
			if g.ActiveLayer == 0 {
				// На первом бублике 0‑й сегмент — дефолтный.
				if g.SelectedSegments[0] == 0 {
					// Дефолтный сегмент: обновляем список доступных Unit/UnitNode.
					g.refreshUnits()
					return
				}

				// Переход на второй бублик возможен только если выбран реальный Unit.
				if len(g.Units.Units) > 0 {
					g.ActiveLayer = 1
					g.SelectedSegments[1] = 0
				}
			} else if g.ActiveLayer == 1 {
				// Стандартное поведение между вторым и третьим бубликом.
				g.ActiveLayer = 2
				g.SelectedSegments[2] = 0
			} else if g.ActiveLayer == 2 {
				unitIdx := g.SelectedSegments[0] - 1
				selectedNodeIdx := g.SelectedSegments[1]
				if unitIdx >= 0 && unitIdx < len(g.Units.Units) {
					selectedUnit := g.Units.Units[unitIdx]
					if selectedNodeIdx < len(selectedUnit.UnitNodes) {
						selectedNode := selectedUnit.UnitNodes[selectedNodeIdx]
						stateData := g.StateData[selectedNode.UUID]
						if g.SelectedSegments[2] == 0 {
							fmt.Println("This is Add button")
							go func() {
								optionName, cancelled := g.AwaitTextInput(true)
								if cancelled {
									return
								}
								optionContent, cancelled := g.AwaitTextInput(false)
								if cancelled {
									return
								}
								if strings.TrimSpace(optionName) == "" {
									return
								}
								_ = g.AddOption(selectedNode.UUID, optionName, optionContent)
							}()
						} else {
							if stateData != nil {
								// Обычный клик по опции - отправляем MQTT сообщение
								fmt.Println(stateData[g.SelectedSegments[2]-1])
								settings := g.PepeClient.GetSettings()
								topicName := settings.PU_DOMAIN + "/" + selectedNode.UUID + "/pepeunit"
								fmt.Println(topicName)
								if g.PepeClient != nil && g.PepeClient.GetMQTTClient() != nil {
									payload := stateData[g.SelectedSegments[2]-1][1]
									g.sendMQTT(topicName, payload)
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

	case ModeTextInput:
		// Обновляем счётчик мигания курсора
		g.CursorTick++

		// ESC отменяет ввод и возвращает назад (в меню/бублик), не закрывая игру.
		// Важно обработать это до Enter, чтобы отмена имела приоритет.
		g.handleKey(ebiten.KeyEscape, func() {
			if g.OnTextInputCancel != nil {
				g.OnTextInputCancel()
			} else {
				g.InputMode = ModeGame
			}
		})

		for _, char := range ebiten.AppendInputChars(nil) {
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

	case ModeHotkeyInput:
		// Обновляем счётчик мигания курсора
		g.CursorTick++

		// ESC отменяет ввод хоткея
		g.handleKey(ebiten.KeyEscape, func() {
			if g.OnHotkeyInputCancel != nil {
				g.OnHotkeyInputCancel()
			} else {
				g.InputMode = ModeGame
			}
		})

		// Захватываем текущую комбинацию клавиш
		currentHotkey := hotkeys.CaptureHotkeyFromEbiten()
		if currentHotkey != "" {
			g.HotkeyInputCurrent = currentHotkey
		}

		// Enter сохраняет хоткей
		g.handleKey(ebiten.KeyEnter, func() {
			if g.OnHotkeyInputDone != nil {
				// Если есть захваченный хоткей, используем его, иначе пустую строку (сброс)
				hotkeyToSave := g.HotkeyInputCurrent
				g.OnHotkeyInputDone(hotkeyToSave)
			}
			g.InputMode = ModeGame
		})

		// Backspace или Delete сбрасывают текущий захваченный хоткей (но не закрывают окно)
		// Пользователь должен нажать ENTER чтобы сохранить пустой хоткей, или ESC чтобы отменить
		if ebiten.IsKeyPressed(ebiten.KeyBackspace) || ebiten.IsKeyPressed(ebiten.KeyDelete) {
			g.HotkeyInputCurrent = ""
		}

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

// StartHotkeyInput переключает игру в режим ввода хоткея для указанной опции
func (g *Game) StartHotkeyInput(unitNodeUUID, optionName string, callback func(string)) {
	g.InputMode = ModeHotkeyInput
	g.HotkeyInputTargetUnitNodeUUID = unitNodeUUID
	g.HotkeyInputTargetOptionName = optionName
	g.HotkeyInputCurrent = ""
	g.OnHotkeyInputDone = callback
	g.OnHotkeyInputCancel = nil
	g.CursorTick = 0
}

// AwaitHotkeyInput блокирует выполнение до получения хоткея
func (g *Game) AwaitHotkeyInput(unitNodeUUID, optionName string) (string, bool) {
	resultChan := make(chan textInputResult, 1)

	g.InputMode = ModeHotkeyInput
	g.HotkeyInputTargetUnitNodeUUID = unitNodeUUID
	g.HotkeyInputTargetOptionName = optionName
	g.HotkeyInputCurrent = ""
	g.CursorTick = 0

	finish := func(res textInputResult) {
		g.OnHotkeyInputDone = nil
		g.OnHotkeyInputCancel = nil
		g.InputMode = ModeGame
		resultChan <- res
		close(resultChan)
	}

	g.OnHotkeyInputDone = func(hotkey string) { finish(textInputResult{text: hotkey, cancelled: false}) }
	g.OnHotkeyInputCancel = func() { finish(textInputResult{text: "", cancelled: true}) }

	res := <-resultChan
	return res.text, res.cancelled
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
				// Первый бублик: дефолтный сегмент + список Units.
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

			// Цвет выбранного Unit для всех бубликов.
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
					// Первый бублик:
					// - все сегменты по умолчанию тёмно‑серые;
					// - 0‑й сегмент "Обновить список юнитов" при наведении
					//   подсвечивается отдельным цветом;
					// - остальные выбранные сегменты подсвечиваются цветом своего Unit.
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
					// Второй и третий бублики:
					// - по умолчанию все сегменты тёмно‑серые;
					// - выбранный сегмент (по индексу SelectedSegments[layerIndex])
					//   подсвечивается цветом выбранного Unit с первого бублика.
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

	// Спинер поверх всего остального.
	g.drawSpinner(screen)

	// Текстовый статус MQTT‑соединения.
	// g.drawMQTTStatus(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	cfg := config.GetConfig()
	return cfg.ScreenWidth, cfg.ScreenHeight
}

// refreshUnits запускает асинхронное обновление списка доступных Unit/UnitNode
// через Pepeunit‑клиент и сбрасывает выбор бубликов на дефолтный сегмент
// после получения результата. При повторном вызове во время выполнения
// предыдущего запроса новый запрос игнорируется.
func (g *Game) refreshUnits() {
	if g.PepeClient == nil || g.refreshInProgress {
		return
	}

	g.refreshInProgress = true
	g.startSpinnerOp()

	client := g.PepeClient

	go func(ch chan<- refreshResult) {
		data, err := FetchUnits(client)
		ch <- refreshResult{data: data, err: err}
	}(g.refreshResultCh)
}

// sendMQTT отправляет MQTT‑сообщение асинхронно и показывает спинер
// на время отправки.
func (g *Game) sendMQTT(topicName, payload string) {
	if g.PepeClient == nil || g.PepeClient.GetMQTTClient() == nil || g.mqttInProgress {
		return
	}

	// Обновляем статус перед началом отправки.
	g.MQTTStatus = "MQTT: sending..."

	g.mqttInProgress = true
	g.startSpinnerOp()

	client := g.PepeClient

	go func(ch chan<- mqttResult) {
		var err error
		if client != nil && client.GetMQTTClient() != nil {
			err = client.GetMQTTClient().Publish(topicName, payload)
		}
		ch <- mqttResult{err: err}
	}(g.mqttResultCh)
}

// resetSelection сбрасывает выбор бубликов и активный слой.
func (g *Game) resetSelection() {
	if len(g.SelectedSegments) >= 1 {
		g.SelectedSegments[0] = 0
	}
	if len(g.SelectedSegments) >= 2 {
		g.SelectedSegments[1] = 0
	}
	if len(g.SelectedSegments) >= 3 {
		g.SelectedSegments[2] = 0
	}
	g.ActiveLayer = 0
}

// readLogEntries читает последние 8 записей из log.json и форматирует их
func (g *Game) readLogEntries() []string {
	// Кэшируем логи, обновляем не чаще раза в секунду
	if time.Since(g.lastLogUpdateTime) < time.Second {
		return g.lastLogEntries
	}

	file, err := os.Open("log.json")
	if err != nil {
		return g.lastLogEntries
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)

	// Читаем файл построчно, так как он может быть большим
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return g.lastLogEntries
	}

	// Парсим JSON объекты из строк
	for _, line := range lines {
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			entries = append(entries, entry)
		}
	}

	// Берем последние 8 записей
	start := len(entries) - 8
	if start < 0 {
		start = 0
	}
	lastEntries := entries[start:]

	// Форматируем записи в нужном формате с ascending order (снизу самые новые)
	var formatted []string
	for _, entry := range lastEntries {
		// Парсим время и форматируем в нужный формат
		parsedTime, err := time.Parse(time.RFC3339, entry.CreateDatetime)
		if err != nil {
			// Если не удалось распарсить, используем оригинальную строку
			formatted = append(formatted, fmt.Sprintf("%s - %s - %s", entry.CreateDatetime[:19], entry.Level, entry.Text))
		} else {
			formatted = append(formatted, fmt.Sprintf("%s - %s - %s", parsedTime.Format("2006-01-02 15:04:05"), entry.Level, entry.Text))
		}
	}

	g.lastLogEntries = formatted
	g.lastLogUpdateTime = time.Now()
	return formatted
}

// startSpinnerOp активирует спинер или добавляет ещё одну операцию,
// требующую его отображения.
func (g *Game) startSpinnerOp() {
	now := time.Now()
	if !g.spinnerActive {
		g.spinnerActive = true
		g.spinnerAngle = 0
		g.spinnerStart = now
		g.spinnerLastUpdate = now
	}
	g.spinnerOpsInFlight++
}

// finishSpinnerOp помечает завершение одной асинхронной операции.
func (g *Game) finishSpinnerOp() {
	if g.spinnerOpsInFlight > 0 {
		g.spinnerOpsInFlight--
	}
}

// updateSpinner обновляет угол поворота спинера и его видимость.
func (g *Game) updateSpinner() {
	if !g.spinnerActive {
		return
	}

	now := time.Now()

	// Обновляем угол поворота: 1 полный оборот в секунду.
	dt := now.Sub(g.spinnerLastUpdate).Seconds()
	if dt < 0 {
		dt = 0
	}
	g.spinnerLastUpdate = now
	g.spinnerAngle += 2 * math.Pi * dt

	// Прячем спинер, если:
	// - нет активных операций
	// - прошло минимум spinnerMinDuration с момента первого запуска
	if g.spinnerOpsInFlight == 0 && now.Sub(g.spinnerStart) >= g.spinnerMinDuration {
		g.spinnerActive = false
	}
}
