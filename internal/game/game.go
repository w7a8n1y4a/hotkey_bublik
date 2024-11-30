package game

import (
	"image/color"
	"math"
	"picker/internal/config"
	"picker/internal/graphics"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Game struct{}

func (g *Game) Update() error {
	mouseX, mouseY := ebiten.CursorPosition()
	dx, dy := mouseX-config.PickerCenterX, mouseY-config.PickerCenterY
	angle := math.Atan2(-float64(dy), -float64(dx)) + math.Pi

	segmentAngle := 2 * math.Pi / float64(config.NumSegments)
	config.SelectedSegment = int(angle / segmentAngle) % config.NumSegments
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if config.BlurredBackground == nil {
		ebitenutil.DebugPrint(screen, "Загрузка размытого фона...")
		return
	}

	screen.DrawImage(config.BlurredBackground, nil)

	segmentAngle := 2 * math.Pi / float64(config.NumSegments)
	for i := 0; i < config.NumSegments; i++ {
		angleStart := float64(i) * segmentAngle
		angleEnd := angleStart + segmentAngle
		clr := color.RGBA{255, 255, 255, 128}
		if i == config.SelectedSegment {
			clr = color.RGBA{255, 0, 0, 200}
		}
		graphics.DrawSegment(screen, config.PickerCenterX, config.PickerCenterY, config.RadiusInner, config.RadiusOuter, angleStart, angleEnd, clr)
	}

	if config.SelectedSegment >= 0 {
		ebitenutil.DebugPrint(screen, "Выбранный сегмент: "+string(rune('A'+config.SelectedSegment)))
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return config.ScreenWidth, config.ScreenHeight
}

