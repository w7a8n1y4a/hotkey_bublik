package game

import (
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"picker/internal/config"

	pepeunit "github.com/w7a8n1y4a/pepeunit_go_client"
)

type UnitNode struct {
	UUID              string `json:"uuid"`
	Type              string `json:"type"`
	VisibilityLevel   string `json:"visibility_level"`
	IsRewritableInput bool   `json:"is_rewritable_input"`
	TopicName         string `json:"topic_name"`
	CreateDatetime    string `json:"create_datetime"`
	State             string `json:"state"`
	UnitUUID          string `json:"unit_uuid"`
	CreatorUUID       string `json:"creator_uuid"`
}

type Unit struct {
	UUID                     string     `json:"uuid"`
	VisibilityLevel          string     `json:"visibility_level"`
	Name                     string     `json:"name"`
	CreateDatetime           string     `json:"create_datetime"`
	IsAutoUpdateFromRepoUnit bool       `json:"is_auto_update_from_repo_unit"`
	RepoBranch               string     `json:"repo_branch"`
	RepoCommit               string     `json:"repo_commit"`
	UnitStateDict            string     `json:"unit_state_dict"`
	CurrentCommitVersion     string     `json:"current_commit_version"`
	LastUpdateDatetime       string     `json:"last_update_datetime"`
	CreatorUUID              string     `json:"creator_uuid"`
	RepoUUID                 string     `json:"repo_uuid"`
	UnitNodes                []UnitNode `json:"unit_nodes"`
}

type UnitsByNodesResponse struct {
	Count int    `json:"count"`
	Units []Unit `json:"units"`
}

type UnitNodesResponse struct {
	Count     int        `json:"count"`
	UnitNodes []UnitNode `json:"unit_nodes"`
}

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
	{0xF4, 0x43, 0x36, 0xFF},
	{0xE9, 0x1E, 0x63, 0xFF},
	{0x9C, 0x27, 0xB0, 0xFF},
	{0x3F, 0x51, 0xB5, 0xFF},
	{0x21, 0x96, 0xF3, 0xFF},
	{0x03, 0xA9, 0xF4, 0xFF},
	{0x00, 0x96, 0x88, 0xFF},
	{0x4C, 0xAF, 0x50, 0xFF},
	{0xFF, 0x98, 0x00, 0xFF},
	{0xFF, 0x57, 0x22, 0xFF},
}

var defaultSegmentColor = color.RGBA{0x42, 0x42, 0x42, 0xFF}

var refreshSegmentColor = color.RGBA{0x60, 0x7D, 0x8B, 0xFF}

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
