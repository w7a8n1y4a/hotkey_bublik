package graphics

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/disintegration/imaging"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kbinani/screenshot"
)

func BlurScreenshot() *ebiten.Image {
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		panic("Ошибка захвата экрана: " + err.Error())
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

var whiteTexture *ebiten.Image

func getWhiteTexture() *ebiten.Image {
	if whiteTexture == nil {
		whiteTexture = ebiten.NewImage(1, 1)
		whiteTexture.Fill(color.White)
	}
	return whiteTexture
}

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

func DrawSegment(screen *ebiten.Image, x, y, rInner, rOuter int, angleStart, angleEnd float64, clr color.Color) {
	drawRingSegment(screen, x, y, rInner, rOuter, angleStart, angleEnd, clr)

	const borderThickness = 2
	if borderThickness <= 0 {
		return
	}

	borderColor := color.RGBA{117, 117, 117, 255}

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

	outerStart := rOuter - borderThickness
	outerEnd := rOuter
	if outerStart < rInner {
		outerStart = rInner
	}
	if outerEnd > outerStart {
		drawRingSegment(screen, x, y, outerStart, outerEnd, angleStart, angleEnd, borderColor)
	}

	const radialBorderAngle = 0.006
	segmentAngle := angleEnd - angleStart
	if segmentAngle <= 0 {
		return
	}

	borderAngle := radialBorderAngle
	if borderAngle*2 > segmentAngle {
		borderAngle = segmentAngle / 4
	}

	leftStart := angleStart
	leftEnd := angleStart + borderAngle
	if leftEnd > angleEnd {
		leftEnd = angleEnd
	}
	if leftEnd > leftStart {
		drawRingSegment(screen, x, y, rInner, rOuter, leftStart, leftEnd, borderColor)
	}

	rightStart := angleEnd - borderAngle
	rightEnd := angleEnd
	if rightStart < angleStart {
		rightStart = angleStart
	}
	if rightEnd > rightStart {
		drawRingSegment(screen, x, y, rInner, rOuter, rightStart, rightEnd, borderColor)
	}
}


