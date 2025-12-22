package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"picker/internal/config"
	"picker/internal/hotkeys"
)

func (g *Game) drawTextInputMessages(screen *ebiten.Image) {
	cfg := config.GetConfig()

	fontFace := LoadFont(24)
	fontBigFace := LoadFont(32)
	fontSize := 24
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
	hintY := cfg.ScreenHeight - fontSize*2
	if hintY < fontSize {
		hintY = fontSize
	}
	DrawCenteredText(
		screen,
		fontFace,
		hintText,
		centerX,
		hintY,
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

	fontSize := 24
	fontFace := LoadFont(float64(fontSize))
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
		color.RGBA{100, 200, 255, 255},
	)

	hintText := "ENTER: сохранить | ESC: отменить | BACKSPACE/DELETE: Сбросить"
	hintY := cfg.ScreenHeight - fontSize*2
	if hintY < fontSize {
		hintY = fontSize
	}
	DrawCenteredText(
		screen,
		fontFace,
		hintText,
		centerX,
		hintY,
		800,
		4,
		color.RGBA{200, 200, 200, 255},
	)
}
