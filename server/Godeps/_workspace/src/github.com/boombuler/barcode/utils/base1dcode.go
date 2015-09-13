// Package utils contain some utilities which are needed to create barcodes
package utils

import (
	"image"
	"image/color"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/boombuler/barcode"
)

type base1DCode struct {
	*BitList
	kind    string
	content string
}

func (c *base1DCode) Content() string {
	return c.content
}

func (c *base1DCode) Metadata() barcode.Metadata {
	return barcode.Metadata{c.kind, 1}
}

func (c *base1DCode) ColorModel() color.Model {
	return color.Gray16Model
}

func (c *base1DCode) Bounds() image.Rectangle {
	return image.Rect(0, 0, c.Len(), 1)
}

func (c *base1DCode) At(x, y int) color.Color {
	if c.GetBit(x) {
		return color.Black
	}
	return color.White
}

// New1DCode creates a new 1D barcode where the bars are represented by the bits in the bars BitList
func New1DCode(codeKind, content string, bars *BitList) barcode.Barcode {
	return &base1DCode{bars, codeKind, content}
}
