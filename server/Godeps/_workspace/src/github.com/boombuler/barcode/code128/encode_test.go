package code128

import (
	"image/color"
	"testing"
)

func testEncode(t *testing.T, txt, testResult string) {
	code, err := Encode(txt)
	if err != nil || code == nil {
		t.Error(err)
	} else {
		if code.Bounds().Max.X != len(testResult) {
			t.Errorf("%v: length missmatch", txt)
		} else {
			for i, r := range testResult {
				if (code.At(i, 0) == color.Black) != (r == '1') {
					t.Errorf("%v: code missmatch on position %d", txt, i)
				}
			}
		}
	}
}

func Test_EncodeFunctionChars(t *testing.T) {
	encFNC1 := "11110101110"
	encFNC2 := "11110101000"
	encFNC3 := "10111100010"
	encFNC4 := "10111101110"
	encStartB := "11010010000"
	encStop := "1100011101011"

	testEncode(t, string(FNC1)+"123", encStartB+encFNC1+"10011100110"+"11001110010"+"11001011100"+"11001000010"+encStop)
	testEncode(t, string(FNC2)+"123", encStartB+encFNC2+"10011100110"+"11001110010"+"11001011100"+"11100010110"+encStop)
	testEncode(t, string(FNC3)+"123", encStartB+encFNC3+"10011100110"+"11001110010"+"11001011100"+"11101000110"+encStop)
	testEncode(t, string(FNC4)+"123", encStartB+encFNC4+"10011100110"+"11001110010"+"11001011100"+"11100011010"+encStop)
}

func Test_Unencodable(t *testing.T) {
	if _, err := Encode(""); err == nil {
		t.Fail()
	}
	if _, err := Encode("ä"); err == nil {
		t.Fail()
	}
}

func Test_EncodeCTable(t *testing.T) {
	testEncode(t, "HI345678H", "110100100001100010100011000100010101110111101000101100011100010110110000101001011110111011000101000111011000101100011101011")
	testEncode(t, "334455", "11010011100101000110001000110111011101000110100100111101100011101011")
}
