package advanced

import (
	"image"
	"math"
)

// CostMap represents the embedding costs for each pixel
type CostMap struct {
	costs  []float64
	width  int
	height int
}

// NewCostMap creates a new cost map for the given image dimensions
func NewCostMap(width, height int) *CostMap {
	return &CostMap{
		costs:  make([]float64, width*height),
		width:  width,
		height: height,
	}
}

// getChannelValue is a helper to get a specific channel value from an RGBA image
func getChannelValue(img *image.RGBA, x, y, channel int) float64 {
	c := img.RGBAAt(x, y)
	switch channel {
	case 0:
		return float64(c.R)
	case 1:
		return float64(c.G)
	case 2:
		return float64(c.B)
	default:
		// Default to Green if channel is invalid
		return float64(c.G)
	}
}

// CalculateCosts computes the embedding costs for each pixel based on edge detection
// of a *specific channel*.
func CalculateCosts(img *image.RGBA, channel int) *CostMap {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	costMap := NewCostMap(width, height)

	// Sobel kernels for edge detection
	sobelX := [3][3]float64{
		{-1, 0, 1},
		{-2, 0, 2},
		{-1, 0, 1},
	}
	sobelY := [3][3]float64{
		{-1, -2, -1},
		{0, 0, 0},
		{1, 2, 1},
	}

	// Calculate gradients and costs for each pixel
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			// Calculate Sobel gradients
			var gx, gy float64
			for i := -1; i <= 1; i++ {
				for j := -1; j <= 1; j++ {
					// Use the specific channel for cost calculation
					pixel := getChannelValue(img, x+i, y+j, channel)
					gx += pixel * sobelX[i+1][j+1]
					gy += pixel * sobelY[i+1][j+1]
				}
			}

			// Calculate gradient magnitude
			gradientMagnitude := math.Sqrt(gx*gx + gy*gy)

			// Calculate cost: higher gradients (edges) = lower cost
			// Add epsilon to prevent division by zero
			cost := 1.0 / (gradientMagnitude + epsilon)

			// Store the cost
			costMap.Set(x, y, cost)
		}
	}

	// Set very high costs for border pixels that we couldn't process
	for y := 0; y < height; y++ {
		costMap.Set(0, y, math.MaxFloat64)
		costMap.Set(width-1, y, math.MaxFloat64)
	}
	for x := 0; x < width; x++ {
		costMap.Set(x, 0, math.MaxFloat64)
		costMap.Set(x, height-1, math.MaxFloat64)
	}

	return costMap
}

// Get returns the cost for the pixel at (x,y)
func (c *CostMap) Get(x, y int) float64 {
	return c.costs[y*c.width + x]
}

// Set sets the cost for the pixel at (x,y)
func (c *CostMap) Set(x, y int, cost float64) {
	c.costs[y*c.width + x] = cost
}

// Width returns the width of the cost map
func (c *CostMap) Width() int {
	return c.width
}

// Height returns the height of the cost map
func (c *CostMap) Height() int {
	return c.height
}