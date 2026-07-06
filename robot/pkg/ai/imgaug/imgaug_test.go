package imgaug

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"testing"

	"google.golang.org/genai"
)

func TestGeminiPromptAugmenter_Augment(t *testing.T) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		t.Skip("GOOGLE_API_KEY not set, skipping integration test")
	}

	ctx := context.Background()
	clientConfig := &genai.ClientConfig{
		APIKey: apiKey,
	}
	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}

	prompt := "In a photorealistic style make the image look like the following scene: warm golden hour, rooftop court skyline backdrop, blue hardcourt, long city shadows.  Perserve the camera angle and position of all the tennis court lines."
	generator, err := NewGeminiImageGenerator(client)
	if err != nil {
		t.Fatalf("Failed to create GeminiPromptAugmenter: %v", err)
	}

	f, err := os.Open("testdata/2025-01-23-eagledale-q4-output-frame-0004.jpg")
	if err != nil {
		t.Fatalf("Failed to open test image: %v", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("Failed to decode image: %v", err)
	}
	// tiles, err := SplitImage2x2(img)
	// if err != nil {
	// 	t.Fatalf("Failed to split image: %v", err)
	// }
	// img = tiles[2]

	// Augment the dummy image
	part, err := ImagePart(img)
	if err != nil {
		t.Fatalf("Augment failed: %v", err)
	}

	augmentedImage, err := generator.Generate(ctx, genai.NewPartFromText(prompt), part)
	if err != nil {
		t.Fatalf("Failed to generate image: %v", err)
	}

	// Save the augmented image to disk for manual verification
	outFile, err := os.Create("testdata/augmented_output.png")
	if err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	err = png.Encode(outFile, augmentedImage)
	if err != nil {
		t.Fatalf("Failed to encode and save augmented image: %v", err)
	}

	t.Logf("Augmented image saved to testdata/augmented_output.png")
}

// func TestGeminiTransfer(t *testing.T) {
// 	apiKey := os.Getenv("GOOGLE_API_KEY")
// 	if apiKey == "" {
// 		t.Skip("GOOGLE_API_KEY not set, skipping integration test")
// 	}

// 	ctx := context.Background()
// 	clientConfig := &genai.ClientConfig{
// 		APIKey: apiKey,
// 	}
// 	client, err := genai.NewClient(ctx, clientConfig)
// 	if err != nil {
// 		t.Fatalf("Failed to create Gemini client: %v", err)
// 	}

// 	prompt := "warm golden hour, rooftop court skyline backdrop, blue hardcourt, long city shadows"
// 	generator, err := NewGeminiImageGenerator(client)
// 	if err != nil {
// 		t.Fatalf("Failed to create GeminiPromptAugmenter: %v", err)
// 	}

// 	f, err := os.Open("testdata/2025-01-23-eagledale-q4-output-frame-0004.jpg")
// 	if err != nil {
// 		t.Fatalf("Failed to open test image: %v", err)
// 	}
// 	defer f.Close()

// 	// Augment the dummy image
// 	part, err := ImagePart(f)
// 	if err != nil {
// 		t.Fatalf("Augment failed: %v", err)
// 	}

// 	augmentedImage, err := generator.Generate(ctx, genai.NewPartFromText(prompt), part)
// 	if err != nil {
// 		t.Fatalf("Failed to generate image: %v", err)
// 	}

// 	// Save the augmented image to disk for manual verification
// 	outFile, err := os.Create("testdata/augmented_output.png")
// 	if err != nil {
// 		t.Fatalf("Failed to create output file: %v", err)
// 	}
// 	defer outFile.Close()

// 	err = png.Encode(outFile, augmentedImage)
// 	if err != nil {
// 		t.Fatalf("Failed to encode and save augmented image: %v", err)
// 	}

// 	t.Logf("Augmented image saved to testdata/augmented_output.png")
// }

// SplitImage2x2 splits the given image into four equal tiles arranged as a 2x2 grid.
// The returned slice is ordered: top-left, top-right, bottom-left, bottom-right.
// If width or height are odd, the extra pixel on the right/bottom is included
// in the corresponding tiles to fully cover the original image without gaps.
func SplitImage2x2(src image.Image) ([]image.Image, error) {
	if src == nil {
		return nil, fmt.Errorf("nil source image")
	}

	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width < 2 || height < 2 {
		return nil, fmt.Errorf("image too small to split: %dx%d", width, height)
	}

	midX := bounds.Min.X + width/2
	midY := bounds.Min.Y + height/2

	// Define the four rectangles. Right/bottom tiles pick up any remainder pixels.
	rectTL := image.Rect(bounds.Min.X, bounds.Min.Y, midX, midY)
	rectTR := image.Rect(midX, bounds.Min.Y, bounds.Max.X, midY)
	rectBL := image.Rect(bounds.Min.X, midY, midX, bounds.Max.Y)
	rectBR := image.Rect(midX, midY, bounds.Max.X, bounds.Max.Y)

	// Helper to copy a sub-rectangle into a new RGBA image starting at (0,0).
	makeTile := func(r image.Rectangle) image.Image {
		dst := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		draw.Draw(dst, dst.Bounds(), src, r.Min, draw.Src)
		return dst
	}

	tiles := []image.Image{
		makeTile(rectTL),
		makeTile(rectTR),
		makeTile(rectBL),
		makeTile(rectBR),
	}
	return tiles, nil
}
