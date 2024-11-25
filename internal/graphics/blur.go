package graphics

import (
	"log"

	"github.com/disintegration/imaging"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kbinani/screenshot"
)

func BlurScreenshot() *ebiten.Image {
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		log.Fatalf("Ошибка захвата экрана: %v", err)
	}

	blurredImg := imaging.Blur(img, 10.0)
	ebitenImg := ebiten.NewImageFromImage(blurredImg)
	return ebitenImg
}

