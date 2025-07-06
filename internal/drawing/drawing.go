package drawing

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

// Draw generates an image with a simple drawing.
func Draw(filePath string) error {
	width := 200
	height := 100

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill the background with white
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.White)
		}
	}

	// Draw a red rectangle
	for x := 50; x < 150; x++ {
		for y := 20; y < 80; y++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}