package main

import (
	"context"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/notnil/tensa/pkg/ai/fusion"
	"github.com/notnil/tensa/pkg/ai/location"
	"github.com/notnil/tensa/pkg/metrics"
	"github.com/notnil/tensa/pkg/tennis/court2d"
)

// Simple Channel PubSub for Demo
type DemoPubSub[T any] struct {
	ch chan T
}

func NewDemoPubSub[T any]() *DemoPubSub[T] {
	return &DemoPubSub[T]{
		ch: make(chan T, 100),
	}
}

func (s *DemoPubSub[T]) Subscribe(ctx context.Context, ch chan<- T) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-s.ch:
			ch <- msg
		}
	}
}

func (s *DemoPubSub[T]) Publish(msg T) {
	select {
	case s.ch <- msg:
	default:
		// Drop if full
	}
}

// Data sent to frontend
type Update struct {
	Type      string  `json:"type"` // "truth", "abs", "rel", "fused"
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Rotation  float64 `json:"rotation"`
	Timestamp int64   `json:"timestamp"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	clients   = make(map[*websocket.Conn]bool)
	clientsMu sync.Mutex
)

func broadcast(u Update) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for client := range clients {
		if err := client.WriteJSON(u); err != nil {
			client.Close()
			delete(clients, client)
		}
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("Starting Fusion Demo")

	// 1. Setup PubSubs
	absPubSub := NewDemoPubSub[metrics.Metric[location.Loc]]()
	relPubSub := NewDemoPubSub[metrics.Metric[location.Loc]]()

	// 2. Setup Fusion Provider
	provider := fusion.NewFusedProvider(logger, absPubSub, relPubSub)

	// 3. Start Fusion Consumer
	fusedCh := make(chan metrics.Metric[location.Loc], 100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := provider.Subscribe(ctx, fusedCh); err != nil {
			logger.Error("Provider subscription failed", "error", err)
		}
	}()

	// 4. Handle Fused Output
	go func() {
		for m := range fusedCh {
			broadcast(Update{
				Type:      "fused",
				X:         m.Value.Location.X,
				Y:         m.Value.Location.Y,
				Rotation:  m.Value.Rotation,
				Timestamp: m.Timestamp.UnixMilli(),
			})
		}
	}()

	// 5. Start Simulation
	go runSimulation(absPubSub, relPubSub)

	// 6. Start Web Server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "pkg/ai/fusion/demo/index.html")
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("Upgrade failed", "error", err)
			return
		}
		clientsMu.Lock()
		clients[conn] = true
		clientsMu.Unlock()
	})

	logger.Info("Server listening on :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		logger.Error("Server failed", "error", err)
	}
}

func runSimulation(
	absPub *DemoPubSub[metrics.Metric[location.Loc]],
	relPub *DemoPubSub[metrics.Metric[location.Loc]],
) {
	ticker := time.NewTicker(10 * time.Millisecond) // 100Hz sim loop
	start := time.Now()

	// Simulation State
	var driftX, driftY, driftRot float64

	// Last broadcast times
	lastAbsTime := time.Now()
	lastRelTime := time.Now()

	// Config
	absInterval := 500 * time.Millisecond
	relInterval := 30 * time.Millisecond

	for range ticker.C {
		now := time.Now()
		t := now.Sub(start).Seconds()

		// Ground Truth: Figure 8
		// Scale factor
		scale := 5.0
		trueX := scale * math.Sin(t/2)
		trueY := scale * math.Sin(t) * math.Cos(t)
		// Heading is tangent to path
		vx := scale * 0.5 * math.Cos(t/2)
		vy := scale * (math.Cos(2 * t)) // derivative of sin(t)cos(t) = cos(2t)
		trueRot := math.Atan2(vy, vx)

		broadcast(Update{
			Type: "truth", X: trueX, Y: trueY, Rotation: trueRot, Timestamp: now.UnixMilli(),
		})

		// Update Drift (Random Walk)
		driftX += (rand.Float64() - 0.5) * 0.06
		driftY += (rand.Float64() - 0.5) * 0.06
		driftRot += (rand.Float64() - 0.5) * 0.03

		// 1. Relative Feed (Fast, Drifting)
		if now.Sub(lastRelTime) >= relInterval {
			// Relative Feed observes Truth + Drift + High Freq Noise
			relNoiseX := (rand.Float64() - 0.5) * 0.02
			relNoiseY := (rand.Float64() - 0.5) * 0.02

			// Simulate sensor in its own frame but drifting.
			obsX := trueX + driftX + relNoiseX
			obsY := trueY + driftY + relNoiseY
			obsRot := trueRot + driftRot

			relLoc := location.Loc{
				Location: court2d.Point{X: obsX, Y: obsY},
				Rotation: obsRot,
			}

			relPub.Publish(metrics.Metric[location.Loc]{
				Timestamp: now,
				Value:     relLoc,
			})

			broadcast(Update{
				Type: "rel", X: obsX, Y: obsY, Rotation: obsRot, Timestamp: now.UnixMilli(),
			})

			lastRelTime = now
		}

		// 2. Absolute Feed (Slow, Noisy but Unbiased)
		if now.Sub(lastAbsTime) >= absInterval {
			// Absolute Feed observes Truth + High Noise (but 0 mean)
			absNoiseX := (rand.Float64() - 0.5) * 1.0 // Large noise
			absNoiseY := (rand.Float64() - 0.5) * 1.0
			absNoiseRot := (rand.Float64() - 0.5) * 0.5

			obsX := trueX + absNoiseX
			obsY := trueY + absNoiseY
			obsRot := trueRot + absNoiseRot

			loc := location.Loc{
				Location: court2d.Point{X: obsX, Y: obsY},
				Rotation: obsRot,
			}

			absPub.Publish(metrics.Metric[location.Loc]{
				Timestamp: now,
				Value:     loc,
			})

			broadcast(Update{
				Type: "abs", X: obsX, Y: obsY, Rotation: obsRot, Timestamp: now.UnixMilli(),
			})

			lastAbsTime = now
		}
	}
}
