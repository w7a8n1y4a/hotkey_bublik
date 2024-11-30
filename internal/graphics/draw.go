package graphics

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

func DrawSegment(screen *ebiten.Image, x, y, rInner, rOuter int, angleStart, angleEnd float64, clr color.Color) {
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

	texture := ebiten.NewImage(1, 1)
	texture.Fill(color.White)
	screen.DrawTriangles(points, indices, texture, nil)
}

