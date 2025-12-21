package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"picker/internal/config"
)

func (g *Game) drawBlurLoadingMessage(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "Загрузка размытого фона...")
}

func (g *Game) drawSpinner(screen *ebiten.Image) {
	if !g.spinnerActive || g.spinnerImage == nil {
		return
	}

	cfg := config.GetConfig()

	b := g.spinnerImage.Bounds()
	w, h := b.Dx(), b.Dy()

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	op.GeoM.Rotate(g.spinnerAngle)
	op.GeoM.Translate(float64(cfg.PickerCenterX), float64(cfg.PickerCenterY))

	screen.DrawImage(g.spinnerImage, op)
}
