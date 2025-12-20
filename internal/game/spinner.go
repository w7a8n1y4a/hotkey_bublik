package game

import (
	"bytes"
	"image"

	_ "embed"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

//go:embed assets/loader.svg
var spinnerSVG []byte

func loadSpinnerImage(size int) (*ebiten.Image, error) {
	if size <= 0 {
		return nil, nil
	}

	icon, err := oksvg.ReadIconStream(bytes.NewReader(spinnerSVG))
	if err != nil {
		return nil, err
	}

	img := image.NewRGBA(image.Rect(0, 0, size, size))

	icon.SetTarget(0, 0, float64(size), float64(size))

	scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
	raster := rasterx.NewDasher(size, size, scanner)

	icon.Draw(raster, 1.0)

	return ebiten.NewImageFromImage(img), nil
}


