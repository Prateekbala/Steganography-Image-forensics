package advanced

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"sort"
)

const (
	headerSize = 8 // Size in bytes for storing message length
)

// AdvancedEncode implements the Edge-Adaptive LSB Matching algorithm
func AdvancedEncode(carrier io.Reader, data io.Reader, result io.Writer) error {
	// 1. Load and prepare image
	img, format, err := getImageAsRGBA(carrier)
	if err != nil {
		return fmt.Errorf("error parsing carrier image: %v", err)
	}

	// Read all data
	dataBytes, err := ioutil.ReadAll(data)
	if err != nil {
		return fmt.Errorf("error reading data: %v", err)
	}

	// 2. Calculate embedding costs
	//    We use the GREEN channel (1) for costs,
	//    because we embed in the RED channel (0).
	//    This prevents the decoder from desyncing.
	bounds := img.Bounds()
	costs := CalculateCosts(img, 1) // 1 = Green Channel

	// 3. Prepare data payload
	header := make([]byte, headerSize)
	binary.BigEndian.PutUint64(header, uint64(len(dataBytes)))
	fullData := append(header, dataBytes...)

	// 4. Check capacity
	capacity := bounds.Dx() * bounds.Dy()
	if len(fullData)*8 > capacity {
		return fmt.Errorf("data is too large for the carrier image: %d bits needed, %d available", len(fullData)*8, capacity)
	}

	// 5. Get flat pixel data (only from the Red channel)
	pixels := make([]byte, capacity)
	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixels[idx] = img.RGBAAt(x, y).R
			idx++
		}
	}

	// 6. Apply optimal changes using LSB Matching
	modifiedPixels := GetOptimalChanges(pixels, fullData, costs)

	// 7. Create result image
	result_img := image.NewRGBA(bounds)
	idx = 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.RGBAAt(x, y)      // Get original G, B, A
			c.R = modifiedPixels[idx]  // Use modified R
			result_img.Set(x, y, c)
			idx++
		}
	}

	// 8. Encode as PNG
	switch format {
	case "png", "jpeg":
		return png.Encode(result, result_img)
	default:
		return fmt.Errorf("unsupported carrier format")
	}
}

// AdvancedDecode extracts the hidden message using the advanced algorithm
func AdvancedDecode(carrier io.Reader, result io.Writer) error {
	// 1. Load and prepare image
	img, _, err := getImageAsRGBA(carrier)
	if err != nil {
		return fmt.Errorf("error parsing carrier image: %v", err)
	}

	// 2. Re-calculate embedding costs
	//    CRITICAL: We MUST use the *exact same* logic as the encoder.
	//    We use the GREEN channel (1), which was not modified.
	bounds := img.Bounds()
	costs := CalculateCosts(img, 1) // 1 = Green Channel

	// 3. Get flat pixel data (only from the Red channel)
	capacity := bounds.Dx() * bounds.Dy()
	pixels := make([]byte, capacity)
	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixels[idx] = img.RGBAAt(x, y).R
			idx++
		}
	}

	// 4. Create a slice of all pixels with their costs
	allPixelCosts := make([]pixelCost, capacity)
	for i := 0; i < capacity; i++ {
		allPixelCosts[i] = pixelCost{
			pos:  i,
			cost: costs.costs[i],
		}
	}

	// 5. Sort the pixels by cost, from lowest to highest
	//    This now perfectly mirrors the encoder's sort order.
	sort.Slice(allPixelCosts, func(i, j int) bool {
		return allPixelCosts[i].cost < allPixelCosts[j].cost
	})

	// 6. Extract the header (first 64 bits)
	if capacity < headerSize*8 {
		return fmt.Errorf("image is too small to contain a header")
	}
	headerBits := make([]byte, headerSize*8)
	for i := 0; i < headerSize*8; i++ {
		pixelPos := allPixelCosts[i].pos
		headerBits[i] = pixels[pixelPos] & 1
	}

	// 7. Convert header bits to bytes
	header := make([]byte, headerSize)
	for i := 0; i < headerSize; i++ {
		for j := 0; j < 8; j++ {
			if headerBits[i*8+j] == 1 {
				header[i] |= 1 << uint(7-j)
			}
		}
	}

	// 8. Get message length
	messageLength := binary.BigEndian.Uint64(header)
	totalHeaderBits := uint64(headerSize * 8)
	totalDataBits := uint64(messageLength * 8)
	totalBits := totalHeaderBits + totalDataBits

	if messageLength == 0 || totalBits > uint64(capacity) {
		return fmt.Errorf("invalid or corrupt message length: %d", messageLength)
	}

	// 9. Extract the actual data bits
	dataBits := make([]byte, totalDataBits)
	for i := 0; i < int(totalDataBits); i++ {
		// Read from the *next* pixels in the sorted cost list
		pixelPos := allPixelCosts[i+int(totalHeaderBits)].pos
		dataBits[i] = pixels[pixelPos] & 1
	}

	// 10. Convert data bits to bytes
	data := make([]byte, messageLength)
	for i := 0; i < int(messageLength); i++ {
		for j := 0; j < 8; j++ {
			if dataBits[i*8+j] == 1 {
				data[i] |= 1 << uint(7-j)
			}
		}
	}

	// 11. Write the extracted data
	_, err = result.Write(data)
	return err
}

func getImageAsRGBA(reader io.Reader) (*image.RGBA, string, error) {
	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, format, fmt.Errorf("error decoding carrier image: %v", err)
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	return rgba, format, nil
}