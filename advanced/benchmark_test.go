package advanced

import (
	"bytes"
	"image"
	"testing"

	"github.com/DimitarPetrov/stegify/steg"
)

func BenchmarkComparison(b *testing.B) {
	// Create test data
	width, height := 1024, 1024
	carrier := image.NewRGBA(image.Rect(0, 0, width, height))
	testData := []byte("This is test data for benchmarking steganography methods!")

	b.Run("Original LSB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := steg.Encode(
				getTestImageReader(carrier),
				bytes.NewReader(testData),
				&buf,
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Advanced LSB Matching + STC", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := AdvancedEncode(
				getTestImageReader(carrier),
				bytes.NewReader(testData),
				&buf,
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// SecurityAnalysis performs statistical tests to compare security
func SecurityAnalysis(t *testing.T) {
	// Create test images
	width, height := 512, 512
	carrier := image.NewRGBA(image.Rect(0, 0, width, height))
	testData := []byte("Test data for security analysis")

	// Original LSB embedding
	var originalBuf bytes.Buffer
	err := steg.Encode(
		getTestImageReader(carrier),
		bytes.NewReader(testData),
		&originalBuf,
	)
	if err != nil {
		t.Fatal(err)
	}
	originalStego, _, _ := image.Decode(bytes.NewReader(originalBuf.Bytes()))

	// Advanced LSB matching
	var advancedBuf bytes.Buffer
	err = AdvancedEncode(
		getTestImageReader(carrier),
		bytes.NewReader(testData),
		&advancedBuf,
	)
	if err != nil {
		t.Fatal(err)
	}
	advancedStego, _, _ := image.Decode(bytes.NewReader(advancedBuf.Bytes()))

	// Perform Chi-Square test on both
	originalChiSquare := calculateChiSquare(originalStego)
	advancedChiSquare := calculateChiSquare(advancedStego)

	t.Logf("Original LSB Chi-Square: %f", originalChiSquare)
	t.Logf("Advanced LSB Chi-Square: %f", advancedChiSquare)

	// A lower Chi-Square value indicates better statistical undetectability
	if advancedChiSquare >= originalChiSquare {
		t.Error("Advanced method did not improve statistical security")
	}
}

// Calculate Chi-Square statistic for an image
func calculateChiSquare(img image.Image) float64 {
	bounds := img.Bounds()
	histogram := make(map[int]int)
	
	// Build histogram
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, _, _, _ := img.At(x, y).RGBA()
			histogram[int(r&1)]++ // Count LSBs
		}
	}

	// Calculate Chi-Square
	expected := float64(bounds.Dx() * bounds.Dy()) / 2 // Expected count for uniform distribution
	chiSquare := 0.0

	for i := 0; i <= 1; i++ {
		observed := float64(histogram[i])
		chiSquare += ((observed - expected) * (observed - expected)) / expected
	}

	return chiSquare
}