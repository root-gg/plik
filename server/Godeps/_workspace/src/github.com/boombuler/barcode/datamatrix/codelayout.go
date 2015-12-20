package datamatrix

import (
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/boombuler/barcode/utils"
)

type setValFunc func(byte)

type codeLayout struct {
	matrix *utils.BitList
	occupy *utils.BitList
	size   *dmCodeSize
}

func newCodeLayout(size *dmCodeSize) *codeLayout {
	result := new(codeLayout)
	result.matrix = utils.NewBitList(size.MatrixColumns() * size.MatrixRows())
	result.occupy = utils.NewBitList(size.MatrixColumns() * size.MatrixRows())
	result.size = size
	return result
}

func (l *codeLayout) Occupied(row, col int) bool {
	return l.occupy.GetBit(col + row*l.size.MatrixColumns())
}

func (l *codeLayout) Set(row, col int, value, bitNum byte, occupy bool) {
	val := ((value >> (7 - bitNum)) & 1) == 1
	if row < 0 {
		row += l.size.MatrixRows()
		col += 4 - ((l.size.MatrixRows() + 4) % 8)
	}
	if col < 0 {
		col += l.size.MatrixColumns()
		row += 4 - ((l.size.MatrixColumns() + 4) % 8)
	}
	l.matrix.SetBit(col+row*l.size.MatrixColumns(), val)
	if occupy {
		l.occupy.SetBit(col+row*l.size.MatrixColumns(), true)
	}
}

func (l *codeLayout) SetSimple(row, col int, value byte) {
	l.Set(row-2, col-2, value, 0, true)
	l.Set(row-2, col-1, value, 1, true)
	l.Set(row-1, col-2, value, 2, true)
	l.Set(row-1, col-1, value, 3, true)
	l.Set(row-1, col-0, value, 4, true)
	l.Set(row-0, col-2, value, 5, true)
	l.Set(row-0, col-1, value, 6, true)
	l.Set(row-0, col-0, value, 7, true)
}

func (l *codeLayout) Corner1(value byte) {
	l.Set(l.size.MatrixRows()-1, 0, value, 0, true)
	l.Set(l.size.MatrixRows()-1, 1, value, 1, true)
	l.Set(l.size.MatrixRows()-1, 2, value, 2, true)
	l.Set(0, l.size.MatrixColumns()-2, value, 3, true)
	l.Set(0, l.size.MatrixColumns()-1, value, 4, true)
	l.Set(1, l.size.MatrixColumns()-1, value, 5, true)
	l.Set(2, l.size.MatrixColumns()-1, value, 6, true)
	l.Set(3, l.size.MatrixColumns()-1, value, 7, true)
}

func (l *codeLayout) Corner2(value byte) {
	l.Set(l.size.MatrixRows()-3, 0, value, 0, true)
	l.Set(l.size.MatrixRows()-2, 0, value, 1, true)
	l.Set(l.size.MatrixRows()-1, 0, value, 2, true)
	l.Set(0, l.size.MatrixColumns()-4, value, 3, true)
	l.Set(0, l.size.MatrixColumns()-3, value, 4, true)
	l.Set(0, l.size.MatrixColumns()-2, value, 5, true)
	l.Set(0, l.size.MatrixColumns()-1, value, 6, true)
	l.Set(1, l.size.MatrixColumns()-1, value, 7, true)
}

func (l *codeLayout) Corner3(value byte) {
	l.Set(l.size.MatrixRows()-3, 0, value, 0, true)
	l.Set(l.size.MatrixRows()-2, 0, value, 1, true)
	l.Set(l.size.MatrixRows()-1, 0, value, 2, true)
	l.Set(0, l.size.MatrixColumns()-2, value, 3, true)
	l.Set(0, l.size.MatrixColumns()-1, value, 4, true)
	l.Set(1, l.size.MatrixColumns()-1, value, 5, true)
	l.Set(2, l.size.MatrixColumns()-1, value, 6, true)
	l.Set(3, l.size.MatrixColumns()-1, value, 7, true)
}

func (l *codeLayout) Corner4(value byte) {
	l.Set(l.size.MatrixRows()-1, 0, value, 0, true)
	l.Set(l.size.MatrixRows()-1, l.size.MatrixColumns()-1, value, 1, true)
	l.Set(0, l.size.MatrixColumns()-3, value, 2, true)
	l.Set(0, l.size.MatrixColumns()-2, value, 3, true)
	l.Set(0, l.size.MatrixColumns()-1, value, 4, true)
	l.Set(1, l.size.MatrixColumns()-3, value, 5, true)
	l.Set(1, l.size.MatrixColumns()-2, value, 6, true)
	l.Set(1, l.size.MatrixColumns()-1, value, 7, true)
}

func (l *codeLayout) IterateSetter() <-chan setValFunc {
	result := make(chan setValFunc)
	go func() {
		row := 4
		col := 0

		for (row < l.size.MatrixRows()) || (col < l.size.MatrixColumns()) {
			if (row == l.size.MatrixRows()) && (col == 0) {
				result <- l.Corner1
			}
			if (row == l.size.MatrixRows()-2) && (col == 0) && (l.size.MatrixColumns()%4 != 0) {
				result <- l.Corner2
			}
			if (row == l.size.MatrixRows()-2) && (col == 0) && (l.size.MatrixColumns()%8 == 4) {
				result <- l.Corner3
			}

			if (row == l.size.MatrixRows()+4) && (col == 2) && (l.size.MatrixColumns()%8 == 0) {
				result <- l.Corner4
			}

			for true {
				if (row < l.size.MatrixRows()) && (col >= 0) && !l.Occupied(row, col) {
					r := row
					c := col
					result <- func(b byte) {
						l.SetSimple(r, c, b)
					}
				}
				row -= 2
				col += 2
				if (row < 0) || (col >= l.size.MatrixColumns()) {
					break
				}
			}

			row += 1
			col += 3

			for true {
				if (row >= 0) && (col < l.size.MatrixColumns()) && !l.Occupied(row, col) {
					r := row
					c := col
					result <- func(b byte) {
						l.SetSimple(r, c, b)
					}
				}
				row += 2
				col -= 2
				if (row >= l.size.MatrixRows()) || (col < 0) {
					break
				}
			}
			row += 3
			col += 1

		}
		close(result)
	}()

	return result
}

func (l *codeLayout) Merge() *datamatrixCode {
	result := newDataMatrixCode(l.size)

	//dotted horizontal lines
	for r := 0; r < l.size.Rows; r += (l.size.RegionRows() + 2) {
		for c := 0; c < l.size.Columns; c += 2 {
			result.set(c, r, true)
		}
	}

	//solid horizontal line
	for r := l.size.RegionRows() + 1; r < l.size.Rows; r += (l.size.RegionRows() + 2) {
		for c := 0; c < l.size.Columns; c++ {
			result.set(c, r, true)
		}
	}

	//dotted vertical lines
	for c := l.size.RegionColumns() + 1; c < l.size.Columns; c += (l.size.RegionColumns() + 2) {
		for r := 1; r < l.size.Rows; r += 2 {
			result.set(c, r, true)
		}
	}

	//solid vertical line
	for c := 0; c < l.size.Columns; c += (l.size.RegionColumns() + 2) {
		for r := 0; r < l.size.Rows; r++ {
			result.set(c, r, true)
		}
	}
	count := 0
	for hRegion := 0; hRegion < l.size.RegionCountHorizontal; hRegion++ {
		for vRegion := 0; vRegion < l.size.RegionCountVertical; vRegion++ {
			for x := 0; x < l.size.RegionColumns(); x++ {
				colMatrix := (l.size.RegionColumns() * hRegion) + x
				colResult := ((2 + l.size.RegionColumns()) * hRegion) + x + 1

				for y := 0; y < l.size.RegionRows(); y++ {
					rowMatrix := (l.size.RegionRows() * vRegion) + y
					rowResult := ((2 + l.size.RegionRows()) * vRegion) + y + 1
					val := l.matrix.GetBit(colMatrix + rowMatrix*l.size.MatrixColumns())
					if val {
						count++
					}

					result.set(colResult, rowResult, val)
				}
			}
		}
	}

	return result
}
