package game

import (
	"bytes"
	"image"
	_ "embed"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed fonts/cornerita_black.ttf
var fontData []byte

//go:embed assets/loader.svg
var spinnerSVG []byte

var (
	baseFont  *opentype.Font
	fontCache = make(map[float64]font.Face)
	fontMu    sync.Mutex
)

func LoadFont(size float64) font.Face {
	fontMu.Lock()
	defer fontMu.Unlock()

	if face, ok := fontCache[size]; ok {
		return face
	}

	if baseFont == nil {
		tt, err := opentype.Parse(fontData)
		if err != nil {
			panic("failed to parse font: " + err.Error())
		}
		baseFont = tt
	}

	face, err := opentype.NewFace(baseFont, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic("failed to create font face: " + err.Error())
	}

	fontCache[size] = face
	return face
}

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


