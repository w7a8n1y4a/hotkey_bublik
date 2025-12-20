package game

import (
	_ "embed"
	"log"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed fonts/cornerita_black.ttf
var fontData []byte

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
			log.Fatalf("failed to parse font: %v", err)
		}
		baseFont = tt
	}

	face, err := opentype.NewFace(baseFont, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatalf("failed to create font face: %v", err)
	}

	fontCache[size] = face
	return face
}
