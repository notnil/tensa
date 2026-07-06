package controller

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/notnil/tensa/pkg/ai/drillsx/api"
	"github.com/notnil/tensa/pkg/ai/drillsx/loader"
	"github.com/notnil/tensa/pkg/ai/location"
	"github.com/notnil/tensa/pkg/ai/navigation"
	"github.com/notnil/tensa/pkg/ai/player"
	"github.com/notnil/tensa/pkg/hware/thrower"
	"github.com/notnil/tensa/pkg/tennis/court2d"
	"github.com/paypal/gatt"
)

// DrillsManager controls lifecycle of a single running drill.
type DrillsManager struct {
	mu      sync.RWMutex
	reg     loader.Registry
	appCtx  context.Context // Application context for cancellation propagation
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
	log     *slog.Logger
	nav     api.Navigator
	mover   api.Mover
}

// NewDrillsManager creates a new drills manager using the provided registry.
func NewDrillsManager(reg loader.Registry, log *slog.Logger, nav api.Navigator, mover api.Mover) *DrillsManager {
	return &DrillsManager{
		reg:   reg,
		log:   log,
		nav:   nav,
		mover: mover,
	}
}

// SetAppContext sets the application context that will be used as the parent context
// for drill execution. This allows drills to be properly cancelled when the application
// shuts down (e.g., via SIGINT/SIGTERM).
func (m *DrillsManager) SetAppContext(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.appCtx = ctx
}

// StartDrill loads the drill by id and starts it, cancelling any existing drill.
// The ctx parameter is used as the parent context for the drill. If the DrillsManager
// has an application context set via SetAppContext, it will be used instead.
func (m *DrillsManager) StartDrill(ctx context.Context, id string, hw Hardware) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("drill id is required")
	}

	// Load drill first to surface errors early
	drill, err := m.reg.GetDrill(id)
	if err != nil {
		return fmt.Errorf("load drill %q: %w", id, err)
	}

	m.mu.Lock()
	// Cancel existing
	if m.running && m.cancel != nil {
		m.log.Info("stopping previous drill to start new one", "newDrillID", id)
		m.cancel()
		m.running = false
		m.cancel = nil
	}
	// Build api runtime using injected navigator and mover
	if m.nav == nil {
		m.mu.Unlock()
		return fmt.Errorf("no navigator configured for drills manager")
	}
	if m.mover == nil {
		m.mu.Unlock()
		return fmt.Errorf("no mover configured for drills manager")
	}
	rt := api.Runtime{
		Nav:            m.nav,
		Mover:          m.mover,
		Thrower:        &apiThrowerAdapter{t: hw.Thrower()},
		Audio:          hw.AudioPlayer(),
		Events:         nil, // TODO: wire up events
		Metrics:        nil, // TODO: wire up metrics
		PlayerProvider: &apiPlayerProviderAdapter{p: hw.PlayerProvider()},
		Log:            m.log,
		Rnd:            rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	// Use application context if available, otherwise use the provided context
	parentCtx := ctx
	if m.appCtx != nil {
		parentCtx = m.appCtx
	}
	m.ctx, m.cancel = context.WithCancel(parentCtx)
	m.running = true
	m.mu.Unlock()

	// Run drill in background
	go func(id string, d api.Drill, rt api.Runtime) {
		m.log.Info("Starting drill", "id", id)
		if err := d.Run(m.ctx, rt); err != nil && m.ctx.Err() == nil {
			m.log.Error("Drill failed", "id", id, "error", err)
		}
		m.mu.Lock()
		m.running = false
		m.cancel = nil
		m.ctx = nil
		m.mu.Unlock()
		m.log.Info("Drill finished", "id", id)
	}(id, drill, rt)

	return nil
}

// StopDrill cancels the current drill if running.
func (m *DrillsManager) StopDrill() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running || m.cancel == nil {
		return fmt.Errorf("no drill is currently running")
	}
	m.cancel()
	m.running = false
	m.cancel = nil
	return nil
}

// IsRunning reports whether a drill is currently running.
func (m *DrillsManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// apiNavigatorAdapter wraps a navigation.Navigator to satisfy api.Navigator,
// converting api.Location to location.Loc.
type apiNavigatorAdapter struct {
	underlying navigation.Navigator
}

func (a apiNavigatorAdapter) Navigate(ctx context.Context, dest api.Location) error {
	loc := location.Loc{
		Location: court2d.Point{X: dest.Point.X, Y: dest.Point.Y},
		Rotation: dest.Rotation,
	}
	return a.underlying.Navigate(ctx, loc)
}

// wrapAPINavigator converts a navigation.Navigator into an api.Navigator.
func wrapAPINavigator(nav navigation.Navigator) api.Navigator {
	if nav == nil {
		return nil
	}
	return apiNavigatorAdapter{underlying: nav}
}

// wrapAPIMover converts a navigation.Mover into an api.Mover.
func wrapAPIMover(mover navigation.Mover) api.Mover {
	if mover == nil {
		return nil
	}
	return apiMoverAdapter{underlying: mover}
}

// apiMoverAdapter wraps navigation.Mover to satisfy api.Mover.
type apiMoverAdapter struct {
	underlying navigation.Mover
}

func (a apiMoverAdapter) Move(dir, speed float64) error {
	return a.underlying.Move(dir, speed)
}

func (a apiMoverAdapter) Rotate(speed float64) error {
	return a.underlying.Rotate(speed)
}

func (a apiMoverAdapter) Stop() error {
	return a.underlying.Stop()
}

// apiThrowerAdapter wraps thrower.Thrower to satisfy api.Thrower by converting settings types.
type apiThrowerAdapter struct {
	t thrower.Thrower
}

func (a *apiThrowerAdapter) Set(s api.Settings) error {
	return a.t.Set(thrower.Settings{Top: s.Top, Bottom: s.Bottom, Angle: s.Angle})
}

func (a *apiThrowerAdapter) Throw(ctx context.Context) error { return a.t.Throw(ctx) }

func (a *apiThrowerAdapter) Load(ctx context.Context) error { return a.t.Load(ctx) }

// apiPlayerProviderAdapter wraps player.Provider to satisfy api.PlayerProvider by converting location types.
type apiPlayerProviderAdapter struct {
	p player.Provider
}

func (a *apiPlayerProviderAdapter) Players() ([]api.Location, error) {
	locs, err := a.p.Players()
	if err != nil {
		return nil, err
	}
	result := make([]api.Location, len(locs))
	for i, loc := range locs {
		result[i] = api.Location{
			Point:    api.Point{X: loc.Location.X, Y: loc.Location.Y},
			Rotation: loc.Rotation,
		}
	}
	return result, nil
}

// BLEDrillStartMessage represents a start command containing a UTF-8 drill ID.
type BLEDrillStartMessage struct {
	ID string
}

func (m *BLEDrillStartMessage) UnmarshalBinary(data []byte) error {
	id := strings.TrimSpace(string(data))
	if id == "" {
		return fmt.Errorf("empty drill id")
	}
	m.ID = id
	return nil
}

// BLEDrillStartHandler handles drill start commands.
type BLEDrillStartHandler struct {
	mgr *DrillsManager
}

func NewBLEDrillStartHandler(mgr *DrillsManager) *BLEDrillStartHandler {
	return &BLEDrillStartHandler{mgr: mgr}
}

func (h *BLEDrillStartHandler) WriteHandler(hw Hardware) func(r gatt.Request, data []byte) (status byte) {
	return func(r gatt.Request, data []byte) (status byte) {
		var msg BLEDrillStartMessage
		if err := msg.UnmarshalBinary(data); err != nil {
			hw.Logger().Error("BLE: failed to parse drill start", "error", err)
			return gatt.StatusUnexpectedError
		}

		// Enable wheels for drill navigation
		if err := hw.Wheels().Enable(); err != nil {
			hw.Logger().Error("BLE: failed to enable wheels for drill", "id", msg.ID, "error", err)
			return gatt.StatusUnexpectedError
		}
		hw.Logger().Info("BLE: wheels enabled for drill", "id", msg.ID)

		// Start, stopping any previous drill
		if err := h.mgr.StartDrill(context.Background(), msg.ID, hw); err != nil {
			hw.Logger().Error("BLE: failed to start drill", "id", msg.ID, "error", err)
			return gatt.StatusUnexpectedError
		}
		hw.Logger().Info("BLE: drill started", "id", msg.ID)
		return gatt.StatusSuccess
	}
}

// BLEDrillStopHandler handles drill stop commands.
type BLEDrillStopHandler struct {
	mgr *DrillsManager
}

func NewBLEDrillStopHandler(mgr *DrillsManager) *BLEDrillStopHandler {
	return &BLEDrillStopHandler{mgr: mgr}
}

func (h *BLEDrillStopHandler) WriteHandler(hw Hardware) func(r gatt.Request, data []byte) (status byte) {
	return func(r gatt.Request, data []byte) (status byte) {
		if err := h.mgr.StopDrill(); err != nil {
			hw.Logger().Error("BLE: failed to stop drill", "error", err)
			return gatt.StatusUnexpectedError
		}

		// Ensure hardware is stopped as well
		if err := hw.Stop(); err != nil {
			hw.Logger().Error("BLE: failed to stop hardware after drill stop", "error", err)
			return gatt.StatusUnexpectedError
		}

		hw.Logger().Info("BLE: drill stopped")
		return gatt.StatusSuccess
	}
}

// BLEDrillCheckHandler handles drill existence checks.
// Write: UTF-8 drill ID
// Read: 1 byte - 0x01 if exists, 0x00 if not
type BLEDrillCheckHandler struct {
	reg         loader.Registry
	mu          sync.Mutex
	lastDrillID string
	lastExists  bool
}

func NewBLEDrillCheckHandler(reg loader.Registry) *BLEDrillCheckHandler {
	return &BLEDrillCheckHandler{reg: reg}
}

func (h *BLEDrillCheckHandler) WriteHandler(hw Hardware) func(r gatt.Request, data []byte) (status byte) {
	return func(r gatt.Request, data []byte) (status byte) {
		if len(data) == 0 {
			hw.Logger().Error("BLE: empty drill check request")
			return gatt.StatusUnexpectedError
		}

		drillID := strings.TrimSpace(string(data))
		if drillID == "" {
			hw.Logger().Error("BLE: empty drill ID in check request")
			return gatt.StatusUnexpectedError
		}

		// Check if drill exists
		_, err := h.reg.GetDrill(drillID)
		exists := err == nil

		h.mu.Lock()
		h.lastDrillID = drillID
		h.lastExists = exists
		h.mu.Unlock()

		hw.Logger().Info("BLE: drill check", "drillID", drillID, "exists", exists)
		return gatt.StatusSuccess
	}
}

func (h *BLEDrillCheckHandler) ReadHandler(hw Hardware) func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
	return func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		h.mu.Lock()
		exists := h.lastExists
		drillID := h.lastDrillID
		h.mu.Unlock()

		// Return 1 byte: 0x01 if exists, 0x00 if not
		result := byte(0x00)
		if exists {
			result = 0x01
		}

		if _, err := rsp.Write([]byte{result}); err != nil {
			hw.Logger().Error("BLE: failed to write drill check response", "error", err)
		} else {
			hw.Logger().Debug("BLE: drill check response", "drillID", drillID, "exists", exists)
		}
	}
}
