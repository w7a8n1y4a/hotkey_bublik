package main

import (
	"image/color"
	"log"
	"math"
	"time"

	"github.com/disintegration/imaging"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/kbinani/screenshot"
	"github.com/micmonay/keybd_event"
)

var (
	screenWidth, screenHeight int
	pickerCenterX, pickerCenterY int
	radiusInner, radiusOuter    = 150, 200
	selectedSegment             = -1
	numSegments                 = 12
	blurredBackground           *ebiten.Image
)

func blurScreenshot() *ebiten.Image {
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		log.Fatalf("Ошибка захвата экрана: %v", err)
	}

	blurredImg := imaging.Blur(img, 10.0)
	ebitenImg := ebiten.NewImageFromImage(blurredImg)
	return ebitenImg
}

type Game struct{}

func (g *Game) Update() error {
	// Получаем положение курсора
	mouseX, mouseY := ebiten.CursorPosition()

	// Рассчитываем угол относительно центра экрана
	dx, dy := mouseX-pickerCenterX, mouseY-pickerCenterY
	angle := math.Atan2(-float64(dy), -float64(dx)) + math.Pi // Инверсия Y для правильного направления

	// Рассчитываем сегмент
	segmentAngle := 2 * math.Pi / float64(numSegments)
	selectedSegment = int(angle / segmentAngle) % numSegments

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if blurredBackground == nil {
		ebitenutil.DebugPrint(screen, "Загрузка размытого фона...")
		return
	}

	screen.DrawImage(blurredBackground, nil)

	// Отрисовка сегментного пикера
	segmentAngle := 2 * math.Pi / float64(numSegments)
	for i := 0; i < numSegments; i++ {
		angleStart := float64(i) * segmentAngle
		angleEnd := angleStart + segmentAngle
		clr := color.RGBA{255, 255, 255, 128}
		if i == selectedSegment {
			clr = color.RGBA{255, 0, 0, 200}
		}
		drawSegment(screen, pickerCenterX, pickerCenterY, radiusInner, radiusOuter, angleStart, angleEnd, clr)
	}

	// Отображение выбранного сегмента
	if selectedSegment >= 0 {
		ebitenutil.DebugPrint(screen, "Выбранный сегмент: "+string(rune('A'+selectedSegment)))
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func drawSegment(screen *ebiten.Image, x, y, rInner, rOuter int, angleStart, angleEnd float64, clr color.Color) {
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

func main() {
	screenWidth, screenHeight = ebiten.ScreenSizeInFullscreen()
	pickerCenterX, pickerCenterY = screenWidth/2, screenHeight/2

	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		log.Fatalf("Ошибка создания горячей клавиши: %v", err)
	}
	kb.SetKeys(keybd_event.VK_C)
	kb.HasCTRL(true)
	kb.HasALT(true)

	go func() {
		for {
			if err := kb.Launching(); err == nil {
				blurredBackground = blurScreenshot()

				ebiten.SetFullscreen(true)
				ebiten.SetWindowTitle("Picker")
				if err := ebiten.RunGame(&Game{}); err != nil {
					log.Fatalf("Ошибка запуска игры: %v", err)
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	select {}
}

