package graphics

import (
	"image"
	"image/draw"
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

	var src image.Image = blurredImg

	nrgba, ok := src.(*image.NRGBA)
	if !ok {
		tmp := image.NewNRGBA(src.Bounds())
		draw.Draw(tmp, tmp.Bounds(), src, src.Bounds().Min, draw.Src)
		nrgba = tmp
	}

	return ebiten.NewImageFromImage(nrgba)
}

