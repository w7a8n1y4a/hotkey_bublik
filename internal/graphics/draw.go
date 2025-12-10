package graphics

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

var whiteTexture *ebiten.Image

// getWhiteTexture возвращает общую белую текстуру 1x1, создавая её один раз лениво.
func getWhiteTexture() *ebiten.Image {
	if whiteTexture == nil {
		whiteTexture = ebiten.NewImage(1, 1)
		whiteTexture.Fill(color.White)
	}
	return whiteTexture
}

// drawRingSegment рисует заполненный сегмент кольца без бордюров.
func drawRingSegment(screen *ebiten.Image, x, y, rInner, rOuter int, angleStart, angleEnd float64, clr color.Color) {
	const steps = 100
	dTheta := (angleEnd - angleStart) / steps
	points := []ebiten.Vertex{}
	indices := []uint16{}

	r, g, b, a := clr.RGBA()

	for i := 0; i <= steps; i++ {
		theta := angleStart + float64(i)*dTheta

		px, py := float32(x)+float32(rOuter)*float32(math.Cos(theta)), float32(y)+float32(rOuter)*float32(math.Sin(theta))
		points = append(points, ebiten.Vertex{
			DstX: px, DstY: py,
			ColorR: float32(r) / 65535,
			ColorG: float32(g) / 65535,
			ColorB: float32(b) / 65535,
			ColorA: float32(a) / 65535,
		})

		px, py = float32(x)+float32(rInner)*float32(math.Cos(theta)), float32(y)+float32(rInner)*float32(math.Sin(theta))
		points = append(points, ebiten.Vertex{
			DstX: px, DstY: py,
			ColorR: float32(r) / 65535,
			ColorG: float32(g) / 65535,
			ColorB: float32(b) / 65535,
			ColorA: float32(a) / 65535,
		})
	}

	for i := 0; i < steps; i++ {
		indices = append(indices, uint16(2*i), uint16(2*i+1), uint16(2*i+2), uint16(2*i+1), uint16(2*i+2), uint16(2*i+3))
	}

	screen.DrawTriangles(points, indices, getWhiteTexture(), nil)
}

// DrawSegment рисует сегмент бублика с тонким бордером по внутреннему и внешнему краю.
func DrawSegment(screen *ebiten.Image, x, y, rInner, rOuter int, angleStart, angleEnd float64, clr color.Color) {
	// Сначала основной заполненный сегмент.
	drawRingSegment(screen, x, y, rInner, rOuter, angleStart, angleEnd, clr)

	// Затем — круговые бордеры. Толщину можно при необходимости подправить.
	const borderThickness = 2
	if borderThickness <= 0 {
		return
	}

	// Цвет бордера — тёмно‑серый, чтобы сегменты чётко отделялись друг от друга,
	// при этом оставался немного светлее основного тёмного фона сегментов.
	borderColor := color.RGBA{117, 117, 117, 255} // #757575

	// Внутренний бордер: небольшое кольцо сразу за внутренним радиусом.
	innerStart := rInner
	innerEnd := rInner + borderThickness
	if innerStart < 0 {
		innerStart = 0
	}
	if innerEnd > rOuter {
		innerEnd = rOuter
	}
	if innerEnd > innerStart {
		drawRingSegment(screen, x, y, innerStart, innerEnd, angleStart, angleEnd, borderColor)
	}

	// Внешний бордер: небольшое кольцо перед внешним радиусом.
	outerStart := rOuter - borderThickness
	outerEnd := rOuter
	if outerStart < rInner {
		outerStart = rInner
	}
	if outerEnd > outerStart {
		drawRingSegment(screen, x, y, outerStart, outerEnd, angleStart, angleEnd, borderColor)
	}

	// Радиальные бордеры: тонкие "кусочки" вдоль границ сегмента по углу.
	// Ширина по углу — фиксированная малая величина, чтобы бордер выглядел
	// одинаково узким на сегментах разного размера.
	const radialBorderAngle = 0.006 // ~0.34°
	segmentAngle := angleEnd - angleStart
	if segmentAngle <= 0 {
		return
	}

	// Ограничиваемся половиной сегмента, чтобы на очень узких сегментах
	// бордеры не перекрывались.
	borderAngle := radialBorderAngle
	if borderAngle*2 > segmentAngle {
		borderAngle = segmentAngle / 4
	}

	// Левая граница сегмента.
	leftStart := angleStart
	leftEnd := angleStart + borderAngle
	if leftEnd > angleEnd {
		leftEnd = angleEnd
	}
	if leftEnd > leftStart {
		drawRingSegment(screen, x, y, rInner, rOuter, leftStart, leftEnd, borderColor)
	}

	// Правая граница сегмента.
	rightStart := angleEnd - borderAngle
	rightEnd := angleEnd
	if rightStart < angleStart {
		rightStart = angleStart
	}
	if rightEnd > rightStart {
		drawRingSegment(screen, x, y, rInner, rOuter, rightStart, rightEnd, borderColor)
	}
}

