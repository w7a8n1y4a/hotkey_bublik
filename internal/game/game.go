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
    cfg := config.GetConfig()
    
	mouseX, mouseY := ebiten.CursorPosition()
	dx, dy := mouseX - cfg.PickerCenterX, mouseY - cfg.PickerCenterY
	angle := math.Atan2(-float64(dy), -float64(dx)) + math.Pi

	segmentAngle := 2 * math.Pi / float64(cfg.NumSegments)

    config.UpdateConfig(func(cfg *config.Config) {
	    cfg.SelectedSegment = int(angle / segmentAngle) % cfg.NumSegments
    })

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    cfg := config.GetConfig()

	if cfg.BlurredBackground == nil {
		ebitenutil.DebugPrint(screen, "Загрузка размытого фона...")
		return
	}

	screen.DrawImage(cfg.BlurredBackground, nil)

	segmentAngle := 2 * math.Pi / float64(cfg.NumSegments)
	for i := 0; i < cfg.NumSegments; i++ {
		angleStart := float64(i) * segmentAngle
		angleEnd := angleStart + segmentAngle
		clr := color.RGBA{255, 255, 255, 128}
		if i == cfg.SelectedSegment {
			clr = color.RGBA{255, 0, 0, 200}
		}
		graphics.DrawSegment(screen, cfg.PickerCenterX, cfg.PickerCenterY, cfg.RadiusInner, cfg.RadiusOuter, angleStart, angleEnd, clr)
	}

	if cfg.SelectedSegment >= 0 {
		ebitenutil.DebugPrint(screen, "Выбранный сегмент: "+string(rune('A'+cfg.SelectedSegment)))
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
    cfg := config.GetConfig()

	return cfg.ScreenWidth, cfg.ScreenHeight
}

