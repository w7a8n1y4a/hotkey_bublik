package game

import (
	_ "embed"
	"image/color"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"picker/internal/config"
)

//go:embed fonts/cornerita_black.ttf
var fontData []byte

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
func DrawCenteredText(screen *ebiten.Image, face font.Face, textContent string, x, y, maxWidth, lineSpacing int, clr color.Color) {
	lines := wrapText(face, textContent, maxWidth)
	totalHeight := len(lines) * (text.BoundString(face, "A").Dy() + lineSpacing)
	startY := y - totalHeight/2

	for i, line := range lines {
		lineWidth := text.BoundString(face, line).Dx()
		startX := x - lineWidth/2
		text.Draw(screen, line, face, startX, startY+(i*(text.BoundString(face, "A").Dy()+lineSpacing)), clr)
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

// drawBlurLoadingMessage выводит сообщение о загрузке размытого фона
func (g *Game) drawBlurLoadingMessage(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "Загрузка размытого фона...")
}

// drawGameModeMessages отвечает за отрисовку подписей сегментов в игровом режиме
func (g *Game) drawGameModeMessages(screen *ebiten.Image, layerIndex int, items [][]string) {
	if len(items) == 0 {
		return
	}

	if g.SelectedSegments[layerIndex] < 0 || len(items) <= g.SelectedSegments[layerIndex] {
		return
	}

	cfg := config.GetConfig()

	fontSize := 24
	fontFace := LoadFont(float64(fontSize))

	centerX := int(cfg.ScreenWidth / 2)
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

	// Основной текст (название элемента)
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

	// Дополнительный текст (значение опции), если он есть
	if len(items[g.SelectedSegments[layerIndex]]) == 2 {
		DrawCenteredText(
			screen,
			fontFace,
			items[g.SelectedSegments[layerIndex]][1],
			centerX,
			centerOption+optionExternalLen+20,
			800,
			4,
			color.White,
		)
	}
}

// drawTextInputMessages выводит подсказки и введённый текст в режиме ввода
func (g *Game) drawTextInputMessages(screen *ebiten.Image) {
	cfg := config.GetConfig()

	fontFace := LoadFont(24)
	fontBigFace := LoadFont(32)
	centerX := cfg.ScreenWidth / 2
	centerY := cfg.ScreenHeight / 2

	targetText := "Write name Option"
	if !g.IsFirstWrite {
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


