package advanced

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"
)

func TestAdvancedEncodeAndDecode(t *testing.T) {
	// Create a test carrier image with pattern
	width, height := 512, 512 // Larger size for more capacity
	carrier := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			carrier.Set(x, y, color.RGBA{
				R: uint8((x * y) % 256),
				G: uint8((x + y) % 256),
				B: uint8((x - y) % 256),
				A: 255,
			})
		}
	}

	// Create test data (smaller than image capacity)
	testData := []byte("This is a test message for the advanced steganography algorithm!")

	// Encode
	var encodedBuf bytes.Buffer
	err := AdvancedEncode(
		getTestImageReader(carrier),
		bytes.NewReader(testData),
		&encodedBuf,
	)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Verify encoded image can be decoded as PNG
	_, _, err = image.Decode(bytes.NewReader(encodedBuf.Bytes()))
	if err != nil {
		t.Fatalf("Failed to decode encoded image: %v", err)
	}

	// Decode hidden data
	var decodedBuf bytes.Buffer
	err = AdvancedDecode(
		bytes.NewReader(encodedBuf.Bytes()),
		&decodedBuf,
	)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	// Compare original and decoded data
	if !bytes.Equal(testData, decodedBuf.Bytes()) {
		t.Errorf("Decoded data does not match original.\nExpected: %v\nGot: %v",
			testData, decodedBuf.Bytes())
	}
}

//
// THIS TEST IS NOW FIXED
//
func TestCostMapCalculation(t *testing.T) {
	// Create a test image with known edges
	width, height := 10, 10
	// Must create an RGBA image, as required by the new function signature
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create a vertical edge in the Green channel (which we will test)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if x < width/2 {
				img.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 255}) // Green = 0
			} else {
				img.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255}) // Green = 255
			}
		}
	}

	// Calculate costs using the Green channel (index 1)
	costs := CalculateCosts(img, 1)

	// Edge pixels should have lower costs
	edgeCost := costs.Get(width/2-1, height/2)
	centerCost := costs.Get(width/4, height/2)

	if edgeCost >= centerCost {
		t.Error("Edge detection failed: edge cost should be lower than center cost")
	}

	// Check that border costs are set to max
	borderCost := costs.Get(0, 0)
	if borderCost != math.MaxFloat64 {
		t.Errorf("Border cost was not set to MaxFloat64, got %f", borderCost)
	}
}

func TestLSBMatching(t *testing.T) {
	// Test embedding bit 1 in pixel with LSB 0
	pixel := byte(100) // 01100100
	bit := byte(1)
	cost := 1.0

	modifiedPixel, _ := LSBMatchingEmbed(pixel, bit, cost)

	// Check that LSB was changed to 1
	if modifiedPixel&1 != 1 {
		t.Error("LSB matching failed to embed bit 1")
	}

	// Check that pixel value only changed by ±1
	diff := int(modifiedPixel) - int(pixel)
	if diff != 1 && diff != -1 {
		t.Error("LSB matching made invalid pixel modification")
	}
}

func getTestImageReader(img image.Image) *bytes.Buffer {
	var buf bytes.Buffer
	// Encode as PNG to create the io.Reader
	if rgbaImg, ok := img.(*image.RGBA); ok {
		png.Encode(&buf, rgbaImg)
	} else {
		// Fallback for other image types, e.g., from test
		png.Encode(&buf, img)
	}
	return &buf
}