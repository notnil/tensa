package main

import (
	"context"
	"fmt"
	"image/jpeg"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"time"

	"github.com/notnil/tensa/pkg/hware/tensax"
)

func startWebServer(ctx context.Context, tensa *tensax.Tensa, addr string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/snapshot", func(w http.ResponseWriter, r *http.Request) {
		handleSnapshot(tensa, w, r)
	})
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		handleStream(ctx, tensa, w, r)
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	tensa.Logger().Info("Starting web server", "addr", addr)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			tensa.Logger().Error("Web server failed", "error", err)
		}
	}()

	<-ctx.Done()
	tensa.Logger().Info("Shutting down web server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)
}

func handleSnapshot(tensa *tensax.Tensa, w http.ResponseWriter, r *http.Request) {
	array := tensa.ZedArray()
	if array == nil {
		http.Error(w, "ZED array not initialized", http.StatusInternalServerError)
		return
	}

	img, err := array.Image()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to capture image: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	if err := jpeg.Encode(w, img.ToRGBA(), &jpeg.Options{Quality: 80}); err != nil {
		tensa.Logger().Error("failed to encode jpeg", "error", err)
	}
}

func handleStream(ctx context.Context, tensa *tensax.Tensa, w http.ResponseWriter, r *http.Request) {
	array := tensa.ZedArray()
	if array == nil {
		http.Error(w, "ZED array not initialized", http.StatusInternalServerError)
		return
	}

	mimeWriter := multipart.NewWriter(w)
	w.Header().Set("Content-Type", fmt.Sprintf("multipart/x-mixed-replace; boundary=%s", mimeWriter.Boundary()))

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.Context().Done():
			return
		default:
			img, err := array.Image()
			if err != nil {
				tensa.Logger().Error("failed to capture image for stream", "error", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			partHeader := make(textproto.MIMEHeader)
			partHeader.Set("Content-Type", "image/jpeg")
			partHeader.Set("Content-Length", fmt.Sprint(img.Size())) // Rough estimate, will be smaller after JPEG

			partWriter, err := mimeWriter.CreatePart(partHeader)
			if err != nil {
				tensa.Logger().Error("failed to create multipart part", "error", err)
				return
			}

			if err := jpeg.Encode(partWriter, img.ToRGBA(), &jpeg.Options{Quality: 70}); err != nil {
				tensa.Logger().Error("failed to encode jpeg for stream", "error", err)
				return
			}

			// Add a small delay to control frame rate if needed, though Image() likely has its own timing
			time.Sleep(30 * time.Millisecond)
		}
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, indexHTML)
}

const indexHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>Tensa ZED Cameras</title>
    <style>
        body { font-family: sans-serif; background: #111; color: #eee; margin: 0; display: flex; flex-direction: column; align-items: center; }
        .controls { padding: 20px; }
        .grid-container { display: grid; grid-template-columns: 1fr; gap: 10px; max-width: 95vw; }
        img { width: 100%; height: auto; border: 2px solid #333; }
        button { padding: 10px 20px; font-size: 16px; cursor: pointer; background: #333; color: #eee; border: 1px solid #555; border-radius: 4px; }
        button:hover { background: #444; }
        .label-grid { position: relative; display: inline-block; }
        .label { position: absolute; background: rgba(0,0,0,0.5); padding: 5px; font-size: 12px; }
        .tl { top: 5px; left: 5px; }
        .tr { top: 5px; right: 5px; }
        .bl { bottom: 5px; left: 5px; }
        .br { bottom: 5px; right: 5px; }
    </style>
</head>
<body>
    <div class="controls">
        <button onclick="setMode('stream')">Live Stream</button>
        <button onclick="setMode('snapshot')">Manual Snapshot</button>
    </div>
    <div class="label-grid">
        <div class="label tl">Back</div>
        <div class="label tr">Right</div>
        <div class="label bl">Front</div>
        <div class="label br">Left</div>
        <img id="cameraGrid" src="/stream" alt="Camera Grid">
    </div>

    <script>
        function setMode(mode) {
            const img = document.getElementById('cameraGrid');
            if (mode === 'stream') {
                img.src = '/stream';
            } else {
                img.src = '/snapshot?t=' + Date.now();
            }
        }
    </script>
</body>
</html>
`
