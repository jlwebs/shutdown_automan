package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/jpeg" // Register JPEG decoder
	"image/png"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: png2ico <input_image> <output.ico>")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	// Read file (whatever format)
	fileData, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Decode to image.Image
	img, _, err := image.Decode(bytes.NewReader(fileData))
	if err != nil {
		fmt.Printf("Error decoding image: %v\n", err)
		os.Exit(1)
	}

	// Re-encode as PNG for ICO embedding
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		fmt.Printf("Error encoding PNG: %v\n", err)
		os.Exit(1)
	}
	pngData := pngBuf.Bytes()

	// Dimensions
	bounds := img.Bounds()
	width := byte(bounds.Dx())
	height := byte(bounds.Dy())
	if bounds.Dx() >= 256 {
		width = 0
	}
	if bounds.Dy() >= 256 {
		height = 0
	}

	// Create ICO file
	outFile, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	// Write ICO Header
	// 0-1: Reserved (0)
	// 2-3: Type (1 for icon)
	// 4-5: Number of images (1)
	header := []byte{0, 0, 1, 0, 1, 0}
	outFile.Write(header)

	// Write Directory Entry (16 bytes)
	entry := make([]byte, 16)
	entry[0] = width
	entry[1] = height
	entry[2] = 0                                                   // Colors
	entry[3] = 0                                                   // Reserved
	binary.LittleEndian.PutUint16(entry[4:], 1)                    // Planes
	binary.LittleEndian.PutUint16(entry[6:], 32)                   // BPP (32 for RGBA)
	binary.LittleEndian.PutUint32(entry[8:], uint32(len(pngData))) // Size
	binary.LittleEndian.PutUint32(entry[12:], 22)                  // Offset (6 header + 16 entry = 22)

	outFile.Write(entry)

	// Write Image Data
	outFile.Write(pngData)

	fmt.Printf("Successfully created %s from %s\n", outputFile, inputFile)
}
