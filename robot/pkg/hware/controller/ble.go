package controller

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/notnil/tensa/pkg/ai/drillsx/loader"
	"github.com/notnil/tensa/pkg/ai/navigation"
	"github.com/paypal/gatt"
)

const (
	ServiceUUID     = "00010000-8786-40ba-ab96-99b91ac981d8"
	MoveUUID        = "00010001-8786-40ba-ab96-99b91ac981d8"
	RotateUUID      = "00010002-8786-40ba-ab96-99b91ac981d8"
	WheelStatusUUID = "00010003-8786-40ba-ab96-99b91ac981d8"
	SetThrowUUID    = "00010004-8786-40ba-ab96-99b91ac981d8"
	ThrowUUID       = "00010005-8786-40ba-ab96-99b91ac981d8"
	WheelEnableUUID = "00010006-8786-40ba-ab96-99b91ac981d8"
	StopUUID        = "00010007-8786-40ba-ab96-99b91ac981d8"
	LoadUUID        = "00010008-8786-40ba-ab96-99b91ac981d8"
	LocationUUID    = "0001000b-8786-40ba-ab96-99b91ac981d8"
	DrillStartUUID  = "00010010-8786-40ba-ab96-99b91ac981d8"
	DrillStopUUID   = "00010011-8786-40ba-ab96-99b91ac981d8"
	DrillStatusUUID = "00010012-8786-40ba-ab96-99b91ac981d8"
	DrillUploadUUID = "00010013-8786-40ba-ab96-99b91ac981d8"
	DrillCheckUUID  = "00010014-8786-40ba-ab96-99b91ac981d8"
	PlayerPoseUUID  = "00010015-8786-40ba-ab96-99b91ac981d8"
	RecordStartUUID = "00010016-8786-40ba-ab96-99b91ac981d8"
	RecordStopUUID  = "00010017-8786-40ba-ab96-99b91ac981d8"
)

type BLEWriter interface {
	WriteHandler(hw Hardware) func(r gatt.Request, data []byte) (status byte)
}

type BLEReader interface {
	ReadHandler(hw Hardware) func(rsp gatt.ResponseWriter, req *gatt.ReadRequest)
}

type BLENotifier interface {
	NotifyHandler(hw Hardware) func(r gatt.Request, n gatt.Notifier)
}

type BLEStopHandler struct{}

// WriteHandler returns a handler function that processes a stop command for the wheels.
// The handler expects an empty payload (or any payload, ignored) and calls hw.Wheels().Stop().
func (BLEStopHandler) WriteHandler(hw Hardware) func(r gatt.Request, data []byte) (status byte) {
	return func(r gatt.Request, data []byte) (status byte) {
		if err := hw.Stop(); err != nil {
			hw.Logger().Error("BLE: failed to command StopAll", "error", err)
			return gatt.StatusUnexpectedError
		}
		return gatt.StatusSuccess
	}
}

// BLEController implements the Controller interface using BLE for input.
type BLEController struct {
	DeviceName      string
	DeviceMAC       string
	Logger          *slog.Logger
	DrillsDirectory string
	// Navigator is injected and used for navigation commands instead of constructing one here.
	Navigator navigation.Navigator
	// Mover is injected and used for low-level movement commands in drills.
	Mover navigation.Mover
}

// Control starts the BLE GATT server and processes BLE commands to control the hardware.
// It blocks until the context is canceled or the BLE server encounters a fatal error.
func (ctrl *BLEController) Control(ctx context.Context, hw Hardware) error {
	var (
		opts []gatt.Option
		err  error
	)
	if strings.TrimSpace(ctrl.DeviceMAC) != "" {
		opts, err = ServerOptionsForMAC(ctrl.DeviceMAC)
		if err != nil {
			return fmt.Errorf("failed to resolve BLE adapter for %s: %w", ctrl.DeviceMAC, err)
		}
	} else {
		opts = DefaultServerOptions
	}

	d, err := gatt.NewDevice(opts...)
	if err != nil {
		return fmt.Errorf("failed to create BLE device: %w", err)
	}
	d.Handle(
		gatt.CentralConnected(func(c gatt.Central) {
			ctrl.Logger.Info("BLE Connected", "id", c.ID())
		}),
		gatt.CentralDisconnected(func(c gatt.Central) {
			ctrl.Logger.Info("BLE Disconnected", "id", c.ID())
			if err := hw.Stop(); err != nil {
				ctrl.Logger.Error("failed to stop hardware", "error", err)
			}
		}),
	)
	service := ctrl.controllerService(ctx, hw)
	d.AddService(service)

	// Start advertising
	d.Init(func(d gatt.Device, s gatt.State) {
		switch s {
		case gatt.StatePoweredOn:
			d.AdvertiseNameAndServices(ctrl.DeviceName, []gatt.UUID{service.UUID()})
		default:
			ctrl.Logger.Info("BLE State", "state", s)
		}
	})
	<-ctx.Done()
	d.StopAdvertising()
	d.StopScanning()
	return nil
}

func (ctrl *BLEController) controllerService(ctx context.Context, hw Hardware) *gatt.Service {
	// Create a new service with a custom UUID for the controller
	s := gatt.NewService(gatt.MustParseUUID(ServiceUUID))
	charMove := s.AddCharacteristic(gatt.MustParseUUID(MoveUUID))
	charMove.HandleWriteFunc(BlEMoveHandler{}.WriteHandler(hw))
	charRotate := s.AddCharacteristic(gatt.MustParseUUID(RotateUUID))
	charRotate.HandleWriteFunc(BLERotateHandler{}.WriteHandler(hw))
	charWheelStatus := s.AddCharacteristic(gatt.MustParseUUID(WheelStatusUUID))
	charWheelStatus.HandleReadFunc(BLEWheelStatusHandler{}.ReadHandler(hw))
	charWheelEnable := s.AddCharacteristic(gatt.MustParseUUID(WheelEnableUUID))
	charWheelEnable.HandleWriteFunc(BLEWheelEnableHandler{}.WriteHandler(hw))
	charSetThrow := s.AddCharacteristic(gatt.MustParseUUID(SetThrowUUID))
	charSetThrow.HandleWriteFunc(BLESetThrowHandler{}.WriteHandler(hw))
	charThrow := s.AddCharacteristic(gatt.MustParseUUID(ThrowUUID))
	charThrow.HandleWriteFunc(BLEThrowHandler{}.WriteHandler(hw))
	charStop := s.AddCharacteristic(gatt.MustParseUUID(StopUUID))
	charStop.HandleWriteFunc(BLEStopHandler{}.WriteHandler(hw))
	charLoad := s.AddCharacteristic(gatt.MustParseUUID(LoadUUID))
	charLoad.HandleWriteFunc(BLELoadHandler{}.WriteHandler(hw))

	charRecordStart := s.AddCharacteristic(gatt.MustParseUUID(RecordStartUUID))
	charRecordStart.HandleWriteFunc(BLERecordStartHandler{}.WriteHandler(hw))

	charRecordStop := s.AddCharacteristic(gatt.MustParseUUID(RecordStopUUID))
	charRecordStop.HandleWriteFunc(BLERecordStopHandler{}.WriteHandler(hw))

	// Location Characteristics - Notification (notify)

	charLocation := s.AddCharacteristic(gatt.MustParseUUID(LocationUUID))
	charLocation.HandleNotifyFunc(BLELocationNotifier{}.NotifyHandler(hw))

	// Drill Characteristics
	drillsDir := ctrl.DrillsDirectory
	if drillsDir == "" {
		drillsDir = "drills"
	}
	drillsRegistry := loader.NewFSRegistry(drillsDir, ctrl.Logger)

	// Drill Status Notifier (for upload feedback)
	drillStatusNotifier := NewBLEDrillStatusNotifier(ctrl.Logger)
	charDrillStatus := s.AddCharacteristic(gatt.MustParseUUID(DrillStatusUUID))
	charDrillStatus.HandleNotifyFunc(drillStatusNotifier.NotifyHandler(hw))

	// Drill Upload Handler
	drillUploadHandler := NewBLEDrillUploadHandler(drillsDir, drillStatusNotifier, ctrl.Logger)
	charDrillUpload := s.AddCharacteristic(gatt.MustParseUUID(DrillUploadUUID))
	charDrillUpload.HandleWriteFunc(drillUploadHandler.WriteHandler(hw))

	// Drill Check Handler (write drill ID, read exists/not exists)
	drillCheckHandler := NewBLEDrillCheckHandler(drillsRegistry)
	charDrillCheck := s.AddCharacteristic(gatt.MustParseUUID(DrillCheckUUID))
	charDrillCheck.HandleWriteFunc(drillCheckHandler.WriteHandler(hw))
	charDrillCheck.HandleReadFunc(drillCheckHandler.ReadHandler(hw))

	// Drill Manager - passes the application context for proper cancellation
	drillsManager := NewDrillsManager(drillsRegistry, ctrl.Logger, wrapAPINavigator(ctrl.Navigator), wrapAPIMover(ctrl.Mover))
	drillsManager.SetAppContext(ctx)
	charDrillStart := s.AddCharacteristic(gatt.MustParseUUID(DrillStartUUID))
	drillStartHandler := NewBLEDrillStartHandler(drillsManager)
	charDrillStart.HandleWriteFunc(drillStartHandler.WriteHandler(hw))

	charDrillStop := s.AddCharacteristic(gatt.MustParseUUID(DrillStopUUID))
	drillStopHandler := NewBLEDrillStopHandler(drillsManager)
	charDrillStop.HandleWriteFunc(drillStopHandler.WriteHandler(hw))

	// Player Pose Characteristic (notify only, 250ms intervals)
	charPlayerPose := s.AddCharacteristic(gatt.MustParseUUID(PlayerPoseUUID))
	charPlayerPose.HandleNotifyFunc(BLEPlayerPoseNotifier{}.NotifyHandler(hw))

	return s
}
