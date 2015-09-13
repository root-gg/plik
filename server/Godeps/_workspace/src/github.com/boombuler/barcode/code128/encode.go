// Package code128 can create Code128 barcodes
package code128

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/boombuler/barcode"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/boombuler/barcode/utils"
)

func strToRunes(str string) []rune {
	result := make([]rune, utf8.RuneCountInString(str))
	i := 0
	for _, r := range str {
		result[i] = r
		i++
	}
	return result
}

func shouldUseCTable(nextRunes []rune, curEncoding byte) bool {
	requiredDigits := 4
	if curEncoding == startCSymbol {
		requiredDigits = 2
	}
	if len(nextRunes) < requiredDigits {
		return false
	}
	for i := 0; i < requiredDigits; i++ {
		if nextRunes[i] < '0' || nextRunes[i] > '9' {
			return false
		}
	}
	return true
}

func getCodeIndexList(content []rune) *utils.BitList {
	result := new(utils.BitList)
	curEncoding := byte(0)
	for i := 0; i < len(content); i++ {

		if shouldUseCTable(content[i:], curEncoding) {
			if curEncoding != startCSymbol {
				if curEncoding == byte(0) {
					result.AddByte(startCSymbol)
				} else {
					result.AddByte(codeCSymbol)
				}
				curEncoding = startCSymbol
			}
			idx := (content[i] - '0') * 10
			i++
			idx = idx + (content[i] - '0')

			result.AddByte(byte(idx))
		} else {
			if curEncoding != startBSymbol {
				if curEncoding == byte(0) {
					result.AddByte(startBSymbol)
				} else {
					result.AddByte(codeBSymbol)
				}
				curEncoding = startBSymbol
			}
			var idx int
			switch content[i] {
			case FNC1:
				idx = 102
				break
			case FNC2:
				idx = 97
				break
			case FNC3:
				idx = 96
				break
			case FNC4:
				idx = 100
				break
			default:
				idx = strings.IndexRune(bTable, content[i])
				break
			}

			if idx < 0 {
				return nil
			}
			result.AddByte(byte(idx))
		}
	}
	return result
}

// Encode creates a Code 128 barcode for the given content
func Encode(content string) (barcode.Barcode, error) {
	contentRunes := strToRunes(content)
	if len(contentRunes) <= 0 || len(contentRunes) > 80 {
		return nil, fmt.Errorf("content length should be between 1 and 80 runes but got %d", len(contentRunes))
	}
	idxList := getCodeIndexList(contentRunes)

	if idxList == nil {
		return nil, fmt.Errorf("\"%s\" could not be encoded", content)
	}

	result := new(utils.BitList)
	sum := 0
	for i, idx := range idxList.GetBytes() {
		if i == 0 {
			sum = int(idx)
		} else {
			sum += i * int(idx)
		}
		result.AddBit(encodingTable[idx]...)
	}
	result.AddBit(encodingTable[sum%103]...)
	result.AddBit(encodingTable[stopSymbol]...)
	return utils.New1DCode("Code 128", content, result), nil
}
