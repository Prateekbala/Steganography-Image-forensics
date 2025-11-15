package advanced

import (
	"crypto/rand"
	"sort"
)

// LSBMatchingEmbed embeds a bit into a pixel value using LSB matching (±1)
// Returns the modified pixel value and the cost of modification
func LSBMatchingEmbed(pixel byte, bit byte, cost float64) (byte, float64) {
	currentLSB := pixel & 1

	// If LSB already matches the bit to embed, no change needed
	if currentLSB == bit {
		return pixel, 0
	}

	// Generate random decision for +1 or -1
	var r [1]byte
	_, err := rand.Read(r[:])
	if err != nil {
		// If we can't get random bytes, default to adding
		// Handle boundary
		if pixel == 255 {
			return pixel - 1, cost
		}
		return pixel + 1, cost
	}
	addOne := r[0]&1 == 1

	// Perform LSB matching
	if addOne {
		if pixel == 255 {
			pixel--
		} else {
			pixel++
		}
	} else {
		if pixel == 0 {
			pixel++
		} else {
			pixel--
		}
	}

	return pixel, cost
}

// pixelCost is a helper struct for sorting pixels by their embedding cost
type pixelCost struct {
	pos  int     // The index of the pixel in the flat pixel array
	cost float64 // The embedding cost
}

// GetOptimalChanges modifies pixels using LSB Matching on the lowest-cost pixels.
// This is the **FIXED** version that sorts pixels by cost and embeds sequentially.
func GetOptimalChanges(img []byte, message []byte, costs *CostMap) []byte {
	result := make([]byte, len(img))
	copy(result, img)

	messageLenBits := len(message) * 8
	if messageLenBits > len(img) {
		// Not enough capacity, though AdvancedEncode should check this first
		return result
	}

	// 1. Create a slice of all pixels with their costs
	allPixelCosts := make([]pixelCost, len(img))
	for i := 0; i < len(img); i++ {
		allPixelCosts[i] = pixelCost{
			pos:  i,
			cost: costs.costs[i],
		}
	}

	// 2. Sort the pixels by cost, from lowest to highest
	sort.Slice(allPixelCosts, func(i, j int) bool {
		return allPixelCosts[i].cost < allPixelCosts[j].cost
	})

	// 3. Embed the message bits into the lowest-cost pixels in order
	for bitIndex := 0; bitIndex < messageLenBits; bitIndex++ {
		// Get the pixel position from the sorted list
		pixelPos := allPixelCosts[bitIndex].pos

		// Get the bit to embed
		byteIndex := bitIndex / 8
		bitOffset := bitIndex % 8
		bitToEmbed := (message[byteIndex] >> (7 - bitOffset)) & 1

		// Modify the pixel in the result image
		result[pixelPos], _ = LSBMatchingEmbed(img[pixelPos], bitToEmbed, costs.costs[pixelPos])
	}

	return result
}
