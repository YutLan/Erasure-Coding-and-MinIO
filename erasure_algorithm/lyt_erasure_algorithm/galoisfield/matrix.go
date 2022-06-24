package galoisfield

import (
	"errors"
	"fmt"
)

type matrix [][]byte

var (
	errInvalidRowSize  = errors.New("invalid row size")
	errInvalidColSize  = errors.New("invalid column size")
	errColSizeMismatch = errors.New("column size is not the same for all rows")
	errMatrixSize      = errors.New("matrix sizes do not match")
	errSingular        = errors.New("matrix is singular")
	errNotSquare       = errors.New("only square matrices can be inverted")
)

func (gf *GF) MatrixMultiply(m, right matrix) (matrix, error) {
	if len(m[0]) != len(right) {
		return nil, fmt.Errorf("columns on left (%d) is different than rows on right (%d)", len(m[0]), len(right))
	}
	result, _ := newMatrix(len(m), len(right[0]))
	for r, row := range result {
		for c := range row {
			var value byte
			for i := range m[0] {
				value ^= gf.Mul(m[r][i], right[i][c])
			}
			result[r][c] = value
		}
	}
	return result, nil
}

func newMatrixData(data [][]byte) (matrix, error) {
	m := matrix(data)
	err := m.Check()
	if err != nil {
		return nil, err
	}
	return m, nil
}

func newMatrix(rows, cols int) (matrix, error) {
	m := matrix(make([][]byte, rows))
	for i := range m {
		m[i] = make([]byte, cols)
	}
	return m, nil
}

func (m matrix) Check() error {
	rows := len(m)
	if rows <= 0 {
		return errInvalidRowSize
	}
	cols := len(m[0])
	if cols <= 0 {
		return errInvalidColSize
	}

	for _, col := range m {
		if len(col) != cols {
			return errColSizeMismatch
		}
	}
	return nil
}

func (m matrix) SubMatrix(rmin, cmin, rmax, cmax int) (matrix, error) {
	result, err := newMatrix(rmax-rmin, cmax-cmin)
	if err != nil {
		return nil, err
	}
	for r := rmin; r < rmax; r++ {
		for c := cmin; c < cmax; c++ {
			result[r-rmin][c-cmin] = m[r][c]
		}
	}
	return result, nil
}

func (m matrix) SwapRows(r1, r2 int) error {
	if r1 < 0 || len(m) <= r1 || r2 < 0 || len(m) <= r2 {
		return errInvalidRowSize
	}
	m[r2], m[r1] = m[r1], m[r2]
	return nil
}

func (m matrix) IsSquare() bool {
	return len(m) == len(m[0])
}

func (m matrix) Augment(right matrix) (matrix, error) {
	if len(m) != len(right) {
		return nil, errMatrixSize
	}

	result, _ := newMatrix(len(m), len(m[0])+len(right[0]))
	for r, row := range m {
		for c := range row {
			result[r][c] = m[r][c]
		}
		cols := len(m[0])
		for c := range right[0] {
			result[r][cols+c] = right[r][c]
		}
	}
	return result, nil
}

func (gf *GF) gaussianElimination(m matrix) error {
	rows := len(m)
	columns := len(m[0])
	for r := 0; r < rows; r++ {
		if m[r][r] == 0 {
			for rowBelow := r + 1; rowBelow < rows; rowBelow++ {
				if m[rowBelow][r] != 0 {
					err := m.SwapRows(r, rowBelow)
					if err != nil {
						return err
					}
					break
				}
			}
		}
		// If we couldn't find one, the matrix is singular.
		if m[r][r] == 0 {
			return errSingular
		}
		// Scale to 1.
		if m[r][r] != 1 {
			scale := gf.Div(1, m[r][r])
			for c := 0; c < columns; c++ {
				m[r][c] = gf.Mul(m[r][c], scale)
			}
		}
		// Make everything below the 1 be a 0 by subtracting
		// a multiple of it.  (Subtraction and addition are
		// both exclusive or in the Galois field.)
		for rowBelow := r + 1; rowBelow < rows; rowBelow++ {
			if m[rowBelow][r] != 0 {
				scale := m[rowBelow][r]
				for c := 0; c < columns; c++ {
					m[rowBelow][c] ^= gf.Mul(scale, m[r][c])
				}
			}
		}
	}

	// Now clear the part above the main diagonal.
	for d := 0; d < rows; d++ {
		for rowAbove := 0; rowAbove < d; rowAbove++ {
			if m[rowAbove][d] != 0 {
				scale := m[rowAbove][d]
				for c := 0; c < columns; c++ {
					m[rowAbove][c] ^= gf.Mul(scale, m[d][c])
				}

			}
		}
	}
	return nil
}

func identityMatrix(size int) (matrix, error) {
	m, err := newMatrix(size, size)
	if err != nil {
		return nil, err
	}
	for i := range m {
		m[i][i] = 1
	}
	return m, nil
}

func (gf *GF) MatrixInvert(m matrix) (matrix, error) {
	if !m.IsSquare() {
		return nil, errNotSquare
	}

	size := len(m)
	work, _ := identityMatrix(size)
	work, _ = m.Augment(work)
	err := gf.gaussianElimination(work)
	if err != nil {
		return nil, err
	}
	return work.SubMatrix(0, size, size, size*2)
}

func (gf *GF) Raid6EncoderMatrix(rows, cols int) (matrix, error) {
	m, err := newMatrix(rows, cols)
	if err != nil {
		return nil, err
	}
	for c := 0; c < cols; c++ {
		m[c][c] ^= 1
	}
	// set the p row
	for c := 0; c < cols; c++ {
		m[rows-2][c] ^= 1
	}
	// set the q row
	for c := 0; c < cols; c++ {
		m[rows-1][c] ^= gf.Power(byte(c+1), 2)
	}
	return m, nil
}

func (gf *GF) Power(a byte, n int) byte {
	res := a
	for i := 1; i < n; i++ {
		res = gf.Mul(res, a)
	}
	return res
}
