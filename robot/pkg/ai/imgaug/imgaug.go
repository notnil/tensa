package imgaug

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"

	"google.golang.org/genai"
)

type GeminiImageGenerator struct {
	prompt string
	client *genai.Client
	model  string
}

func NewGeminiImageGenerator(client *genai.Client) (*GeminiImageGenerator, error) {
	return &GeminiImageGenerator{
		client: client,
		model:  "gemini-2.5-flash-image-preview",
	}, nil
}

func (a *GeminiImageGenerator) Generate(ctx context.Context, parts ...*genai.Part) (image.Image, error) {
	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}
	result, err := a.client.Models.GenerateContent(
		ctx,
		a.model,
		contents,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("gemini failed to generate content: %w", err)
	}

	var imgData []byte
	if result != nil && len(result.Candidates) > 0 && result.Candidates[0] != nil && result.Candidates[0].Content != nil {
		for _, part := range result.Candidates[0].Content.Parts {
			if part.InlineData != nil && len(part.InlineData.Data) > 0 {
				imgData = part.InlineData.Data
				break
			}
		}
	}
	if len(imgData) == 0 {
		return nil, fmt.Errorf("no image data returned by model")
	}

	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	return img, nil
}

func ImagePartFromReader(r io.Reader) (*genai.Part, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}
	mime := http.DetectContentType(b)
	part := genai.NewPartFromBytes(b, mime)
	return part, nil
}

func ImagePart(img image.Image) (*genai.Part, error) {
	buf := bytes.NewBuffer(nil)
	err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 95})
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}
	part := genai.NewPartFromBytes(buf.Bytes(), "image/jpeg")
	return part, nil
}
