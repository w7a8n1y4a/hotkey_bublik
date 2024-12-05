package game

import (
	"fmt"
	_ "embed"
    "github.com/atotto/clipboard"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
    "github.com/hajimehoshi/ebiten/v2/text"
	"image/color"
    "strings"
	"math"
    "log"
    "golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"picker/internal/config"
	"picker/internal/graphics"
	"picker/internal/mqttclient"
	"picker/internal/queries"
	"picker/internal/state"
)

type InputMode int

const (
	ModeGame InputMode = iota
	ModeTextInput
)

//go:embed fonts/cornerita_black.ttf
var fontData []byte

type Game struct {
	Client          *mqttclient.MqttClient
	Units           queries.UnitsByNodesResponse
	StateManager    *state.StateManager
	KeyDownMap      map[ebiten.Key]bool // Состояние кнопок
	SelectedSegments []int              // Хранение текущего выбора для каждого слоя
	ActiveLayer     int                 // Индекс текущего слоя
    InputMode       InputMode
	TextInput       string
	OnTextInputDone func(string)
    IsFirstWrite    bool
}

// LoadFont загружает шрифт из файла
func LoadFont(size float64) font.Face {

	tt, err := opentype.Parse(fontData)
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}

	face, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatalf("failed to create font face: %v", err)
	}

	return face
}

// DrawCenteredText отрисовывает большой текст с центрированием
func DrawCenteredText(screen *ebiten.Image, face font.Face, textContent string, x, y, maxWidth, lineSpacing int, color color.Color) {
	lines := wrapText(face, textContent, maxWidth)
	totalHeight := len(lines) * (text.BoundString(face, "A").Dy() + lineSpacing)
	startY := y - totalHeight/2

	for i, line := range lines {
		lineWidth := text.BoundString(face, line).Dx()
		startX := x - lineWidth/2
		text.Draw(screen, line, face, startX, startY+(i*(text.BoundString(face, "A").Dy()+lineSpacing)), color)
	}
}

// wrapText разбивает текст на строки, которые помещаются в указанную ширину
func wrapText(face font.Face, textContent string, maxWidth int) []string {
	words := strings.Fields(textContent)
	lines := []string{}
	line := ""

	for _, word := range words {
		testLine := line + " " + word
		if text.BoundString(face, strings.TrimSpace(testLine)).Dx() > maxWidth {
			lines = append(lines, strings.TrimSpace(line))
			line = word
		} else {
			line = testLine
		}
	}
	lines = append(lines, strings.TrimSpace(line))

	return lines
}

// Метод для переключения в режим ввода текста
func (g *Game) StartTextInput(callback func(string)) {
	g.InputMode = ModeTextInput
	g.TextInput = ""
	g.OnTextInputDone = callback
}

func (g *Game) AwaitTextInput(isFirstWrite bool) string {
    // Создаем канал для передачи текста
    resultChan := make(chan string)
    
    // Переключаем игру в режим ввода текста
    g.InputMode = ModeTextInput
    g.TextInput = ""
    g.IsFirstWrite = isFirstWrite
    
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
                        if g.SelectedSegments[2] != 0 && g.SelectedSegments[2]-1 < len(stateData) {
                            optionName := stateData[g.SelectedSegments[2]-1][0]
                            // Используем uгRemoveOption для удаления опции
                            err := g.StateManager.RemoveOption(selectedNode.UUID, optionName)
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
                        stateData := g.StateManager.GetState()[selectedNode.UUID]
                        if g.SelectedSegments[2] == 0 {
                            fmt.Println("This is Add button")
                            go func() {
                                optionName := g.AwaitTextInput(true)
                                optionContent := g.AwaitTextInput(false)
                                g.StateManager.AddOption(selectedNode.UUID, optionName, optionContent)
                            }()
                        } else {
                            if stateData != nil{
                            
                                fmt.Println(stateData[g.SelectedSegments[2]-1])
                                // TODO: change /pepeunit logic to adaptive without /pepeunit
                                topicName := cfg.PEPEUNIT_URL + "/" + selectedNode.UUID + "/pepeunit"
                                fmt.Println(topicName)
                                err := g.Client.Publish(topicName, 0, false, stateData[g.SelectedSegments[2]-1][1]) 
                                if err == nil {
                                    fmt.Println("Sendet")
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
	
        for _, char := range ebiten.InputChars() {
            if char != '\n' && char != '\r' {
                g.TextInput += string(char)
            }
        }

        // Обработка Backspace
        if len(g.TextInput) > 0 {
            g.handleKey(ebiten.KeyBackspace, func() {
                g.TextInput = g.TextInput[:len(g.TextInput)-1]
            })
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

func (g *Game) Draw(screen *ebiten.Image) {

	cfg := config.GetConfig()
	if cfg.BlurredBackground == nil {
		ebitenutil.DebugPrint(screen, "Загрузка размытого фона...")
		return
	}
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
                        stateData := g.StateManager.GetState()[selectedNode.UUID]
                        
                        items = append(items, []string{"Create New Option"})

                        for _, value := range stateData {
                            items = append(items, value)
                        }
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
                    angleStart+0.01 - 0.0015 * float64(layerIndex),
                    angleEnd-0.01 + 0.0015 * float64(layerIndex),
                    clr,
                )
            }
            if g.SelectedSegments[layerIndex] >= 0 && len(items) > g.SelectedSegments[layerIndex] {
                var fontSize int = 24
                var centerY int = 0
                fontFace := LoadFont(float64(fontSize)) // Укажите путь и размер шрифта

                centerX := int(cfg.ScreenWidth/2)
                centerUnit := int(cfg.ScreenHeight/2) - int(float64(fontSize)/2)
                centerUnitNode := int(cfg.ScreenHeight/2) + int(float64(fontSize) * 1.5)
                centerOption := int(cfg.ScreenHeight/2)

                optionExternalLen := int(float64(cfg.RadiusInner) + float64(cfg.ThickSegment) * 3 + float64(fontSize) * float64(layerIndex))

               
                switch layerIndex {
                    case 0:
                        centerY = centerUnit
                    case 1:
                        centerY = centerUnitNode
                    case 2:
                        centerY = centerOption - optionExternalLen + fontSize
                }
                // _, _, _ = fontFace, centerX, centerY
                // fmt.Println(items[g.SelectedSegments[layerIndex]]) 
                DrawCenteredText(
                    screen,
                    fontFace,
                    items[g.SelectedSegments[layerIndex]][0],
                    centerX,
                    centerY,
                    cfg.RadiusInner,
                    4,
                    color.White,
                )

                if len(items[g.SelectedSegments[layerIndex]]) == 2 {
                    DrawCenteredText(
                        screen,
                        fontFace,
                        items[g.SelectedSegments[layerIndex]][1],
                        centerX,
                        centerOption + optionExternalLen + 20,
                        800,
                        4,
                        color.White,
                    )
 
                }

            }
        }
    case ModeTextInput:
        fontFace := LoadFont(24) // Укажите путь и размер шрифта
        fontBigFace := LoadFont(32) // Укажите путь и размер шрифта
        centerX := cfg.ScreenWidth/2
        centerY := cfg.ScreenHeight/2
        
        var targetText string = "Write name Option"

        if g.IsFirstWrite != true {
            targetText = "Write UnitNode state"
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
            "Enter text or <CTRL + V>",
            centerX,
            centerY/2,
            300,
            4,
            color.White,
        )

        DrawCenteredText(
            screen,
            fontFace,
            g.TextInput,
            centerX,
            centerY,
            800,
            4,
            color.White,
        )

	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	cfg := config.GetConfig()
	return cfg.ScreenWidth, cfg.ScreenHeight
}

