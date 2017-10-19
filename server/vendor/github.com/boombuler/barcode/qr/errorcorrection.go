package qr

import (
	"github.com/boombuler/barcode/utils"
)

type errorCorrection struct {
	fld       *utils.GaloisField
	polynomes []*utils.GFPoly
}

var ec = newGF()

func newGF() *errorCorrection {
	fld := utils.NewGaloisField(285)

	return &errorCorrection{fld,
		[]*utils.GFPoly{
			utils.NewGFPoly(fld, []byte{1}),
		},
	}
}

func (ec *errorCorrection) getPolynomial(degree int) *utils.GFPoly {
	if degree >= len(ec.polynomes) {
		last := ec.polynomes[len(ec.polynomes)-1]
		for d := len(ec.polynomes); d <= degree; d++ {
			next := last.Multiply(utils.NewGFPoly(ec.fld, []byte{1, byte(ec.fld.ALogTbl[d-1])}))
			ec.polynomes = append(ec.polynomes, next)
			last = next
		}
	}
	return ec.polynomes[degree]
}

func (ec *errorCorrection) calcECC(data []byte, eccCount byte) []byte {
	generator := ec.getPolynomial(int(eccCount))
	info := utils.NewGFPoly(ec.fld, data)
	info = info.MultByMonominal(int(eccCount), 1)

	_, remainder := info.Divide(generator)

	result := make([]byte, eccCount)
	numZero := int(eccCount) - len(remainder.Coefficients)
	copy(result[numZero:], remainder.Coefficients)
	return result
}
