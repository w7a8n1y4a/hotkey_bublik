package game

import (
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"picker/internal/config"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

type InputMode int

const (
	ModeGame InputMode = iota
	ModeTextInput
	ModeHotkeyInput
)

type LogEntry struct {
	CreateDatetime string `json:"create_datetime"`
	Level          string `json:"level"`
	Text           string `json:"text"`
}

var unitColors = []color.RGBA{
	{0xF4, 0x43, 0x36, 0xFF}, // 0 → красный      (#F44336)
	{0xE9, 0x1E, 0x63, 0xFF}, // 1 → розовый      (#E91E63)
	{0x9C, 0x27, 0xB0, 0xFF}, // 2 → фиолетовый   (#9C27B0)
	{0x3F, 0x51, 0xB5, 0xFF}, // 3 → индиго       (#3F51B5)
	{0x21, 0x96, 0xF3, 0xFF}, // 4 → синий        (#2196F3)
	{0x03, 0xA9, 0xF4, 0xFF}, // 5 → голубой      (#03A9F4)
	{0x00, 0x96, 0x88, 0xFF}, // 6 → бирюзовый    (#009688)
	{0x4C, 0xAF, 0x50, 0xFF}, // 7 → зелёный      (#4CAF50)
	{0xFF, 0x98, 0x00, 0xFF}, // 8 → оранжевый    (#FF9800)
	{0xFF, 0x57, 0x22, 0xFF}, // 9 → тёплый оранж (#FF5722)
}

var defaultSegmentColor = color.RGBA{0x42, 0x42, 0x42, 0xFF} // #424242

var refreshSegmentColor = color.RGBA{0x60, 0x7D, 0x8B, 0xFF} // #607D8B

type Game struct {
	PepeClient                    *pepeunit.PepeunitClient
	Units                         UnitsByNodesResponse
	OnHotkeysChanged              func()
	StateData                     map[string][][]string
	KeyDownMap                    map[ebiten.Key]bool
	CursorTick                    int
	BackspaceFrames               int
	SelectedSegments              []int
	ActiveLayer                   int
	InputMode                     InputMode
	TextInput                     string
	OnTextInputDone               func(string)
	OnTextInputCancel             func()
	IsFirstWrite                  bool
	HotkeyInputTargetUnitNodeUUID string
	HotkeyInputTargetOptionName   string
	HotkeyInputCurrent            string
	OnHotkeyInputDone             func(string)
	OnHotkeyInputCancel           func()
	lastNodeInfoJSON              string
	lastNodeUnitIdx               int
	lastNodeUnitNodeIdx           int

	lastLogEntries    []string
	lastLogUpdateTime time.Time

	spinnerImage       *ebiten.Image
	spinnerActive      bool
	spinnerAngle       float64
	spinnerStart       time.Time
	spinnerLastUpdate  time.Time
	spinnerOpsInFlight int
	spinnerMinDuration time.Duration

	refreshResultCh   chan refreshResult
	refreshInProgress bool
	mqttResultCh      chan mqttResult
	mqttInProgress    bool

	MQTTStatus string
}

type refreshResult struct {
	data UnitsByNodesResponse
	err  error
}

type mqttResult struct {
	err error
}

type textInputResult struct {
	text      string
	cancelled bool
}

func NewGame(client *pepeunit.PepeunitClient, data UnitsByNodesResponse, stateData map[string][][]string) (*Game, error) {
	cfg := config.GetConfig()

	spinnerSize := 2 * (cfg.RadiusInner - 40)
	if spinnerSize < 10 {
		spinnerSize = 10
	}

	spinnerImg, err := loadSpinnerImage(spinnerSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load spinner image: %w", err)
	}

	mqttStatus := "MQTT: disabled"
	if client != nil && client.GetMQTTClient() != nil {
		mqttStatus = "MQTT: ready"
	}

	g := &Game{
		PepeClient:         client,
		Units:              data,
		StateData:          stateData,
		KeyDownMap:         make(map[ebiten.Key]bool),
		SelectedSegments:   make([]int, 3),
		ActiveLayer:        0,
		spinnerImage:       spinnerImg,
		spinnerMinDuration: 100 * time.Millisecond,
		refreshResultCh:    make(chan refreshResult, 1),
		mqttResultCh:       make(chan mqttResult, 1),
		MQTTStatus:         mqttStatus,
	}

	return g, nil
}

func (g *Game) GetState() map[string][][]string {
	copyState := make(map[string][][]string)
	for uuid, options := range g.StateData {
		dup := make([][]string, len(options))
		for i, pair := range options {
			dup[i] = append([]string{}, pair...)
		}
		copyState[uuid] = dup
	}
	return copyState
}
