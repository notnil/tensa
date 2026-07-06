package tensax

import (
	"github.com/notnil/tensa/pkg/hware/thrower/clearcore"
	"github.com/notnil/tensa/pkg/tennis/court2d"
	"github.com/notnil/tensa/pkg/util/jsonx"
)

// Config represents the main configuration structure.
type Config struct {
	Audio           Audio      `json:"audio"`
	Wheels          Wheels     `json:"wheels"`
	Thrower         Thrower    `json:"thrower"`
	Zed             Zed        `json:"zed"`
	Player          Player     `json:"player"`
	Location        Location   `json:"location"`
	Navigation      Navigation `json:"navigation"`
	Logging         Logging    `json:"logging"`
	Stats           Stats      `json:"stats"`
	BLEDeviceName   string     `json:"ble_device_name"`
	BLEDeviceMAC    string     `json:"ble_device_mac,omitempty"`
	DrillsDirectory string     `json:"drills_directory,omitempty"`
}

type Logging struct {
	Level     int    `json:"level"`
	Directory string `json:"directory"`
	LogStats  bool   `json:"log_stats"`
	LogWheels bool   `json:"log_wheels"`
}

type Audio struct {
	SoundEffectsEnabled bool `json:"sound_effects_enabled"`
}

type Location struct {
	Type   string `json:"type"`
	Filter bool   `json:"filter"`
}

type Zed struct {
	Type               string          `json:"type"`
	Resolution         int             `json:"resolution"`
	FPS                int             `json:"fps"`
	DepthMode          int             `json:"depth_mode"`
	MaxExposureTime    jsonx.Duration  `json:"max_exposure_time,omitempty"`
	SerialNumbers      map[string]uint `json:"serial_numbers"`
	RecordingDirectory string          `json:"recording_directory,omitempty"`
}

type Player struct {
	Type string `json:"type"`
}

type Stats struct {
	Type    string `json:"type"`
	Subject string `json:"subject"`
}

// Wheels represents the wheels configuration.
type Wheels struct {
	Type               string         `json:"type"`
	VendorID           string         `json:"vendor_id"`
	MoveAcceleration   jsonx.Duration `json:"move_acceleration"`
	MoveDeceleration   jsonx.Duration `json:"move_deceleration"`
	RotateAcceleration jsonx.Duration `json:"rotate_acceleration"`
	RotateDeceleration jsonx.Duration `json:"rotate_deceleration"`
	SyncPeriod         jsonx.Duration `json:"sync_period"`
	CommandTimeout     jsonx.Duration `json:"command_timeout"`
}

// Thrower represents the thrower configuration.
type Thrower struct {
	Type       string           `json:"type"`
	TCPAddress string           `json:"tcp_address"`
	ClearCore  clearcore.Config `json:"clear_core"`
}

type Navigation struct {
	Type     string   `json:"type"`
	TwoStage TwoStage `json:"two_stage"`
	Subject  string   `json:"subject"`
}

type TwoStage struct {
	Translation  Translation    `json:"translation"`
	Rotation     Rotation       `json:"rotation"`
	RestDuration jsonx.Duration `json:"rest_duration"`
	Timeout      jsonx.Duration `json:"timeout"`
}

type Translation struct {
	FarSpeed    float64         `json:"far_speed"`
	NearSpeed   float64         `json:"near_speed"`
	OnThreshold float64         `json:"on_threshold"`
	SafeZone    court2d.Polygon `json:"safe_zone"`
	Timeout     jsonx.Duration  `json:"timeout"`
}

type Rotation struct {
	MaxSpeed    float64        `json:"max_speed"`
	MinSpeed    float64        `json:"min_speed"`
	OnThreshold float64        `json:"on_threshold"`
	Timeout     jsonx.Duration `json:"timeout"`
}
