package code128

var encodingTable = [107][]bool{
	[]bool{true, true, false, true, true, false, false, true, true, false, false},
	[]bool{true, true, false, false, true, true, false, true, true, false, false},
	[]bool{true, true, false, false, true, true, false, false, true, true, false},
	[]bool{true, false, false, true, false, false, true, true, false, false, false},
	[]bool{true, false, false, true, false, false, false, true, true, false, false},
	[]bool{true, false, false, false, true, false, false, true, true, false, false},
	[]bool{true, false, false, true, true, false, false, true, false, false, false},
	[]bool{true, false, false, true, true, false, false, false, true, false, false},
	[]bool{true, false, false, false, true, true, false, false, true, false, false},
	[]bool{true, true, false, false, true, false, false, true, false, false, false},
	[]bool{true, true, false, false, true, false, false, false, true, false, false},
	[]bool{true, true, false, false, false, true, false, false, true, false, false},
	[]bool{true, false, true, true, false, false, true, true, true, false, false},
	[]bool{true, false, false, true, true, false, true, true, true, false, false},
	[]bool{true, false, false, true, true, false, false, true, true, true, false},
	[]bool{true, false, true, true, true, false, false, true, true, false, false},
	[]bool{true, false, false, true, true, true, false, true, true, false, false},
	[]bool{true, false, false, true, true, true, false, false, true, true, false},
	[]bool{true, true, false, false, true, true, true, false, false, true, false},
	[]bool{true, true, false, false, true, false, true, true, true, false, false},
	[]bool{true, true, false, false, true, false, false, true, true, true, false},
	[]bool{true, true, false, true, true, true, false, false, true, false, false},
	[]bool{true, true, false, false, true, true, true, false, true, false, false},
	[]bool{true, true, true, false, true, true, false, true, true, true, false},
	[]bool{true, true, true, false, true, false, false, true, true, false, false},
	[]bool{true, true, true, false, false, true, false, true, true, false, false},
	[]bool{true, true, true, false, false, true, false, false, true, true, false},
	[]bool{true, true, true, false, true, true, false, false, true, false, false},
	[]bool{true, true, true, false, false, true, true, false, true, false, false},
	[]bool{true, true, true, false, false, true, true, false, false, true, false},
	[]bool{true, true, false, true, true, false, true, true, false, false, false},
	[]bool{true, true, false, true, true, false, false, false, true, true, false},
	[]bool{true, true, false, false, false, true, true, false, true, true, false},
	[]bool{true, false, true, false, false, false, true, true, false, false, false},
	[]bool{true, false, false, false, true, false, true, true, false, false, false},
	[]bool{true, false, false, false, true, false, false, false, true, true, false},
	[]bool{true, false, true, true, false, false, false, true, false, false, false},
	[]bool{true, false, false, false, true, true, false, true, false, false, false},
	[]bool{true, false, false, false, true, true, false, false, false, true, false},
	[]bool{true, true, false, true, false, false, false, true, false, false, false},
	[]bool{true, true, false, false, false, true, false, true, false, false, false},
	[]bool{true, true, false, false, false, true, false, false, false, true, false},
	[]bool{true, false, true, true, false, true, true, true, false, false, false},
	[]bool{true, false, true, true, false, false, false, true, true, true, false},
	[]bool{true, false, false, false, true, true, false, true, true, true, false},
	[]bool{true, false, true, true, true, false, true, true, false, false, false},
	[]bool{true, false, true, true, true, false, false, false, true, true, false},
	[]bool{true, false, false, false, true, true, true, false, true, true, false},
	[]bool{true, true, true, false, true, true, true, false, true, true, false},
	[]bool{true, true, false, true, false, false, false, true, true, true, false},
	[]bool{true, true, false, false, false, true, false, true, true, true, false},
	[]bool{true, true, false, true, true, true, false, true, false, false, false},
	[]bool{true, true, false, true, true, true, false, false, false, true, false},
	[]bool{true, true, false, true, true, true, false, true, true, true, false},
	[]bool{true, true, true, false, true, false, true, true, false, false, false},
	[]bool{true, true, true, false, true, false, false, false, true, true, false},
	[]bool{true, true, true, false, false, false, true, false, true, true, false},
	[]bool{true, true, true, false, true, true, false, true, false, false, false},
	[]bool{true, true, true, false, true, true, false, false, false, true, false},
	[]bool{true, true, true, false, false, false, true, true, false, true, false},
	[]bool{true, true, true, false, true, true, true, true, false, true, false},
	[]bool{true, true, false, false, true, false, false, false, false, true, false},
	[]bool{true, true, true, true, false, false, false, true, false, true, false},
	[]bool{true, false, true, false, false, true, true, false, false, false, false},
	[]bool{true, false, true, false, false, false, false, true, true, false, false},
	[]bool{true, false, false, true, false, true, true, false, false, false, false},
	[]bool{true, false, false, true, false, false, false, false, true, true, false},
	[]bool{true, false, false, false, false, true, false, true, true, false, false},
	[]bool{true, false, false, false, false, true, false, false, true, true, false},
	[]bool{true, false, true, true, false, false, true, false, false, false, false},
	[]bool{true, false, true, true, false, false, false, false, true, false, false},
	[]bool{true, false, false, true, true, false, true, false, false, false, false},
	[]bool{true, false, false, true, true, false, false, false, false, true, false},
	[]bool{true, false, false, false, false, true, true, false, true, false, false},
	[]bool{true, false, false, false, false, true, true, false, false, true, false},
	[]bool{true, true, false, false, false, false, true, false, false, true, false},
	[]bool{true, true, false, false, true, false, true, false, false, false, false},
	[]bool{true, true, true, true, false, true, true, true, false, true, false},
	[]bool{true, true, false, false, false, false, true, false, true, false, false},
	[]bool{true, false, false, false, true, true, true, true, false, true, false},
	[]bool{true, false, true, false, false, true, true, true, true, false, false},
	[]bool{true, false, false, true, false, true, true, true, true, false, false},
	[]bool{true, false, false, true, false, false, true, true, true, true, false},
	[]bool{true, false, true, true, true, true, false, false, true, false, false},
	[]bool{true, false, false, true, true, true, true, false, true, false, false},
	[]bool{true, false, false, true, true, true, true, false, false, true, false},
	[]bool{true, true, true, true, false, true, false, false, true, false, false},
	[]bool{true, true, true, true, false, false, true, false, true, false, false},
	[]bool{true, true, true, true, false, false, true, false, false, true, false},
	[]bool{true, true, false, true, true, false, true, true, true, true, false},
	[]bool{true, true, false, true, true, true, true, false, true, true, false},
	[]bool{true, true, true, true, false, true, true, false, true, true, false},
	[]bool{true, false, true, false, true, true, true, true, false, false, false},
	[]bool{true, false, true, false, false, false, true, true, true, true, false},
	[]bool{true, false, false, false, true, false, true, true, true, true, false},
	[]bool{true, false, true, true, true, true, false, true, false, false, false},
	[]bool{true, false, true, true, true, true, false, false, false, true, false},
	[]bool{true, true, true, true, false, true, false, true, false, false, false},
	[]bool{true, true, true, true, false, true, false, false, false, true, false},
	[]bool{true, false, true, true, true, false, true, true, true, true, false},
	[]bool{true, false, true, true, true, true, false, true, true, true, false},
	[]bool{true, true, true, false, true, false, true, true, true, true, false},
	[]bool{true, true, true, true, false, true, false, true, true, true, false},
	[]bool{true, true, false, true, false, false, false, false, true, false, false},
	[]bool{true, true, false, true, false, false, true, false, false, false, false},
	[]bool{true, true, false, true, false, false, true, true, true, false, false},
	[]bool{true, true, false, false, false, true, true, true, false, true, false, true, true},
}

// const startASymbol byte = 103
const startBSymbol byte = 104
const startCSymbol byte = 105

const codeBSymbol byte = 100
const codeCSymbol byte = 99

const stopSymbol byte = 106

const (
	// FNC1 - Special Function 1
	FNC1 = '\u00f1'
	// FNC2 - Special Function 2
	FNC2 = '\u00f2'
	// FNC3 - Special Function 3
	FNC3 = '\u00f3'
	// FNC4 - Special Function 4
	FNC4 = '\u00f4'
)

const bTable = " !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"
