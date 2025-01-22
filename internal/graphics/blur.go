package graphics

import (
	"log"
    "image"
	"github.com/disintegration/imaging"
	"github.com/kbinani/screenshot"
)

func BlurScreenshot() *image.NRGBA {
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		log.Fatalf("Ошибка захвата экрана: %v", err)
	}

	blurredImg := imaging.Blur(img, 10.0)
	return blurredImg
}

