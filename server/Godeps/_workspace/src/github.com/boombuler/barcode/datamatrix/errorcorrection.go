package datamatrix

import (
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/boombuler/barcode/utils"
)

type errorCorrection struct {
	fld       *utils.GaloisField
	polynomes map[int][]int
}

var ec *errorCorrection = newErrorCorrection()

func newErrorCorrection() *errorCorrection {
	result := new(errorCorrection)
	result.fld = utils.NewGaloisField(301)
	result.polynomes = make(map[int][]int)
	return result
}

func (ec *errorCorrection) getPolynomial(count int) []int {
	poly, ok := ec.polynomes[count]
	if !ok {
		idx := 1
		poly = make([]int, count+1)
		poly[0] = 1
		for i := 1; i <= count; i++ {
			poly[i] = 1
			for j := i - 1; j > 0; j-- {
				if poly[j] != 0 {
					poly[j] = ec.fld.ALogTbl[(int(ec.fld.LogTbl[poly[j]])+idx)%255]
				}
				poly[j] = ec.fld.AddOrSub(poly[j], poly[j-1])
			}
			poly[0] = ec.fld.ALogTbl[(int(ec.fld.LogTbl[poly[0]])+idx)%255]
			idx++
		}
		poly = poly[0:count]
		ec.polynomes[count] = poly
	}
	return poly
}

func (ec *errorCorrection) calcECCBlock(data []byte, poly []int) []byte {
	ecc := make([]byte, len(poly)+1)

	for i := 0; i < len(data); i++ {
		k := ec.fld.AddOrSub(int(ecc[0]), int(data[i]))
		for j := 0; j < len(ecc)-1; j++ {
			ecc[j] = byte(ec.fld.AddOrSub(int(ecc[j+1]), ec.fld.Multiply(k, poly[len(ecc)-j-2])))
		}
	}
	return ecc
}

func (ec *errorCorrection) calcECC(data []byte, size *dmCodeSize) []byte {
	buff := make([]byte, size.DataCodewordsPerBlock())
	poly := ec.getPolynomial(size.ErrorCorrectionCodewordsPerBlock())

	dataSize := len(data)
	// make some space for error correction codes
	data = append(data, make([]byte, size.ECCCount)...)

	for block := 0; block < size.BlockCount; block++ {
		// copy the data for the current block to buff
		j := 0
		for i := block; i < dataSize; i += size.BlockCount {
			buff[j] = data[i]
			j++
		}
		// calc the error correction codes
		ecc := ec.calcECCBlock(buff, poly)
		// and append them to the result
		j = 0
		for i := block; i < size.ErrorCorrectionCodewordsPerBlock()*size.BlockCount; i += size.BlockCount {
			data[dataSize+i] = ecc[j]
			j++
		}
	}

	return data
}
