package advanced

import (
	"crypto/rand"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"math"
)

// MediaType represents different types of cover media
type MediaType int

const (
	ImageRGB MediaType = iota
	ImageGrayscale
	ImageYCbCr
)

// CoverMedia provides an interface for different types of cover media
type CoverMedia interface {
	// GetSize returns the capacity of the cover media in bytes
	GetSize() int64
	
	// GetCosts returns the embedding costs for each position
	GetCosts() []float64
	
	// Embed embeds data at given positions
	Embed(data []byte, positions []int) error
	
	// Extract extracts data from given positions
	Extract(positions []int) ([]byte, error)
	
	// Save writes the media to an output stream
	Save(w io.Writer) error
}

// RGBImage implements CoverMedia for RGB images
type RGBImage struct {
	img    *image.RGBA
	costs  []float64
}

// NewRGBImage creates a new RGB image cover media
func NewRGBImage(img image.Image) (*RGBImage, error) {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	rgbImg := &RGBImage{
		img:    rgba,
		costs:  make([]float64, bounds.Dx()*bounds.Dy()*3), // RGB channels
	}

	// Calculate costs using edge detection
	rgbImg.calculateCosts()

	return rgbImg, nil
}

func (r *RGBImage) GetSize() int64 {
	bounds := r.img.Bounds()
	return int64(bounds.Dx() * bounds.Dy() * 3) // 3 channels
}

func (r *RGBImage) GetCosts() []float64 {
	return r.costs
}

func (r *RGBImage) Embed(data []byte, positions []int) error {
	if len(positions) < len(data)*8 {
		return fmt.Errorf("insufficient positions for data")
	}

	bounds := r.img.Bounds()
	width := bounds.Dx()
	
	for i, pos := range positions {
		if i >= len(data)*8 {
			break
		}

		byteIndex := i / 8
		bitIndex := i % 8
		bit := (data[byteIndex] >> uint(7-bitIndex)) & 1

		// Calculate x, y, and channel from position
		x := (pos / 3) % width
		y := (pos / 3) / width
		channel := pos % 3

		c := r.img.RGBAAt(x, y)
		switch channel {
		case 0:
			c.R = modifyPixelLSBMatching(c.R, bit)
		case 1:
			c.G = modifyPixelLSBMatching(c.G, bit)
		case 2:
			c.B = modifyPixelLSBMatching(c.B, bit)
		}
		r.img.SetRGBA(x, y, c)
	}

	return nil
}

func (r *RGBImage) Extract(positions []int) ([]byte, error) {
	dataLen := len(positions) / 8
	data := make([]byte, dataLen)
	bounds := r.img.Bounds()
	width := bounds.Dx()

	for i, pos := range positions {
		if i >= len(positions) {
			break
		}

		byteIndex := i / 8
		bitIndex := i % 8

		x := (pos / 3) % width
		y := (pos / 3) / width
		channel := pos % 3

		c := r.img.RGBAAt(x, y)
		var bit byte
		switch channel {
		case 0:
			bit = c.R & 1
		case 1:
			bit = c.G & 1
		case 2:
			bit = c.B & 1
		}

		data[byteIndex] |= bit << uint(7-bitIndex)
	}

	return data, nil
}

func (r *RGBImage) Save(w io.Writer) error {
	return png.Encode(w, r.img)
}

func (r *RGBImage) calculateCosts() {
	bounds := r.img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Sobel operators
	sobelX := [3][3]float64{{-1, 0, 1}, {-2, 0, 2}, {-1, 0, 1}}
	sobelY := [3][3]float64{{-1, -2, -1}, {0, 0, 0}, {1, 2, 1}}

	// Calculate gradients for each channel
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			// Process each channel
			for c := 0; c < 3; c++ {
				var gradX, gradY float64

				// Apply Sobel operators
				for i := -1; i <= 1; i++ {
					for j := -1; j <= 1; j++ {
						pixel := r.getChannelValue(x+i, y+j, c)
						gradX += pixel * sobelX[i+1][j+1]
						gradY += pixel * sobelY[i+1][j+1]
					}
				}

				// Calculate gradient magnitude
				gradMag := math.Sqrt(gradX*gradX + gradY*gradY)
				
				// Convert to cost (inverse relationship)
				cost := 1.0 / (gradMag + epsilon)
				
				// Store cost
				pos := (y*width + x) * 3 + c
				r.costs[pos] = cost
			}
		}
	}

	// Set high costs for border pixels
	for y := 0; y < height; y++ {
		for c := 0; c < 3; c++ {
			r.costs[(y*width+0)*3+c] = math.MaxFloat64
			r.costs[(y*width+width-1)*3+c] = math.MaxFloat64
		}
	}
	for x := 0; x < width; x++ {
		for c := 0; c < 3; c++ {
			r.costs[(0*width+x)*3+c] = math.MaxFloat64
			r.costs[((height-1)*width+x)*3+c] = math.MaxFloat64
		}
	}
}

func (r *RGBImage) getChannelValue(x, y, channel int) float64 {
	c := r.img.RGBAAt(x, y)
	switch channel {
	case 0:
		return float64(c.R)
	case 1:
		return float64(c.G)
	case 2:
		return float64(c.B)
	default:
		return 0
	}
}

// modifyPixelLSBMatching modifies pixel value using LSB matching (±1)
func modifyPixelLSBMatching(pixel, bit byte) byte {
	if pixel&1 == bit {
		return pixel
	}

	// Randomly add or subtract 1
	if pixel == 0 {
		return 1
	} else if pixel == 255 {
		return 254
	}

	// Generate random choice
	if randBool() {
		return pixel + 1
	}
	return pixel - 1
}

// Helper function to generate random boolean
func randBool() bool {
	var r [1]byte
	rand.Read(r[:])
	return r[0]&1 == 1
}