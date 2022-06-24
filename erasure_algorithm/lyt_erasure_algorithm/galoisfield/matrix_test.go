package galoisfield

import (
	"fmt"
	"testing"
)

func TestNewMatrix(t *testing.T) {
	row := 3
	col := 4
	rNewMatrix, err := newMatrix(row, col)
	if err != nil {
		t.Errorf("Test error: creating Matrix")
	}
	fmt.Println("Succecss", rNewMatrix)
}

func TestMatrixMultiply(t *testing.T) {
	m1, err := newMatrixData(
		[][]byte{
			{1, 1},
			{1, 1},
		})
	m2, err := newMatrixData(
		[][]byte{
			{1, 1},
			{1, 1},
		})

	field := fields[0]
	rMulMatrix, err := field.MatrixMultiply(m1, m2)
	if err != nil {
		t.Errorf("Test error: adding Matrix")
	}
	fmt.Println("Succecss", rMulMatrix)
}

func TestMartrixInverse(t *testing.T) {
	m, err := newMatrixData(
		[][]byte{
			{2, 0, 0},
			{0, 1, 0},
			{0, 0, 1},
		})
	field := fields[0]
	inverseMatrix, err := field.MatrixInvert(m)
	//inverseMatrix, err = field.MatrixInvert(inverseMatrix)
	if err != nil {
		t.Errorf("Test error: matroxInverser")
	}
	fmt.Println("succuess", inverseMatrix)
}

func TestRaid6EncoderMatrix(t *testing.T) {
	field := fields[0]
	m, err := field.Raid6EncoderMatrix(5, 3)
	if err != nil {
		t.Errorf("Test error: creating Raid6 EncoderMatrix")
	}
	fmt.Println("Encoder Matrix: ", m)
	subMatrix, err := m.SubMatrix(2, 0, 5, 3)
	fmt.Println("SubMatrix is: ", subMatrix)
	inverseMatrix, err := field.MatrixInvert(subMatrix)
	//inverseMatrix, err = field.MatrixInvert(inverseMatrix)
	if err != nil {
		t.Errorf("Test error: matroxInverser")
	}
	fmt.Println("succuess", inverseMatrix)
}

func TestRaid6LossReconstruct(t *testing.T) {
	field := fields[0]
	print(field.Size())
	m, _ := field.Raid6EncoderMatrix(5, 3)

	subMatrix, _ := m.SubMatrix(2, 0, 5, 3)
	decodeMatrix, _ := field.MatrixInvert(subMatrix)

	// Encoder
	data := [][]byte{
		{1, 2, 3, 4},
		{4, 5, 6, 4},
		{7, 8, 9, 4}}
	fmt.Println("Original data:", data)
	encoderResult, err := field.MatrixMultiply(m, data)
	if err != nil {
		t.Errorf("Testing errors: multiplay the encoder matrix")
	}
	fmt.Println("Encoder result:", encoderResult)

	// loss
	subEncoderResult, _ := encoderResult.SubMatrix(2, 0, 5, 4)
	reconstruct_data, _ := field.MatrixMultiply(decodeMatrix, subEncoderResult)

	fmt.Println("Reconstruct data:", reconstruct_data)

}
