package config

import "github.com/hajimehoshi/ebiten/v2"

var (
	ScreenWidth, ScreenHeight int
	PickerCenterX, PickerCenterY int
	RadiusInner, RadiusOuter      = 150, 200
	SelectedSegment               = -1
	NumSegments                   = 12
	BlurredBackground             *ebiten.Image
)

