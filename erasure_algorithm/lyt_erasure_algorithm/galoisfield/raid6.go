package galoisfield

import (
	"errors"
	"fmt"
	"io"
)

var (
	ErrParityNonEqualTwo   = errors.New("The parity shards is not equal to 2")
	ErrInvShardNum         = errors.New("cannot create Encoder with less than one data shard or less than zero parity shards")
	ErrMaxShardNum         = errors.New("cannot create Encoder with more than the field hards")
	ErrShardNoData         = errors.New("no shard data")
	ErrShardSize           = errors.New("shard sizes do not match")
	ErrTooFewShards        = errors.New("too few shards given")
	ErrShortData           = errors.New("not enough data to fill the number of requested shards")
	ErrReconstructRequired = errors.New("reconstruction required as one or more required data shards are nil")
)

type Encoder interface {
	Encode(shards [][]byte) error
	Reconstruct(shards [][]byte) error
	ReconstructData(shards [][]byte) error
	// Update(shards [][]byte, newDatashards [][]byte) error
	Split(data []byte) ([][]byte, error)
	// Join(dst io.Writer, shards [][]byte, outSize int) error
}

type Raid6 struct {
	DataShards   int //
	ParityShards int // 2 Number of parity shards, should not be modified.
	Shards       int // Total number of shards. It should be DataShards + 1
	m            matrix
	field        *GF
}

func Raid6New(dataShards, parityShards int) (Encoder, error) {
	r := Raid6{
		DataShards:   dataShards,
		ParityShards: parityShards,
		Shards:       dataShards + parityShards,
	}
	if parityShards != 2 {
		return nil, ErrParityNonEqualTwo
	}

	if dataShards <= 0 || parityShards < 0 {
		return nil, ErrInvShardNum
	}
	if dataShards+parityShards > 256 {
		return nil, ErrMaxShardNum
	}

	if parityShards == 0 {
		return &r, nil
	}

	r.field = Poly84320_g2
	r.m, _ = r.field.Raid6EncoderMatrix(r.Shards, r.DataShards)

	return &r, nil
}

// An array 'shards' containing data shards followed by parity shards.
func (r *Raid6) Encode(shards [][]byte) error {
	fmt.Println("encoderShards:", shards)
	if len(shards) != r.Shards {
		return ErrShardNoData
	}
	if len(shards[0]) != len(shards[r.DataShards]) {
		return ErrShardSize
	}
	data := shards[0:r.DataShards]
	encoderResult, _ := r.field.MatrixMultiply(r.m, data)
	fmt.Println("encoderResult:", encoderResult)
	copy(shards, encoderResult)

	return nil
}

func (r *Raid6) ReconstructData(shards [][]byte) error {
	if len(shards) != r.Shards {
		return ErrTooFewShards
	}
	subShards := make([][]byte, r.DataShards)
	validIndices := make([]int, r.DataShards)
	invalidIndices := make([]int, 0)
	subMatrixRow := 0
	for matrixRow := 0; matrixRow < r.Shards && subMatrixRow < r.DataShards; matrixRow++ {
		if len(shards[matrixRow]) != 0 {
			subShards[subMatrixRow] = shards[matrixRow]
			validIndices[subMatrixRow] = matrixRow
			subMatrixRow++
		} else {
			invalidIndices = append(invalidIndices, matrixRow)
		}
	}

	subMatrix, _ := newMatrix(r.DataShards, r.DataShards)
	for subMatrixRow, validIndex := range validIndices {
		for c := 0; c < r.DataShards; c++ {
			subMatrix[subMatrixRow][c] = r.m[validIndex][c]
		}
	}
	dataDecodeMatrix, err := r.field.MatrixInvert(subMatrix)
	if err != nil {
		return err
	}
	// reconstructData, err := r.field.MatrixMultiply(dataDecodeMatrix, subShards)
	// copy(shards, reconstructData)
	// fmt.Println("DecoderResult data:", shards)

	reconstructData, err := r.field.MatrixMultiply(dataDecodeMatrix, subShards)
	data := reconstructData[0:r.DataShards]
	encoderResult, _ := r.field.MatrixMultiply(r.m, data)
	copy(shards, encoderResult)
	fmt.Println("DecoderResult:", shards)
	return err
}

func (r *Raid6) Reconstruct(shards [][]byte) error {
	if len(shards) != r.Shards {
		return ErrTooFewShards
	}
	subShards := make([][]byte, r.DataShards)
	validIndices := make([]int, r.DataShards)
	invalidIndices := make([]int, 0)
	subMatrixRow := 0
	for matrixRow := 0; matrixRow < r.Shards && subMatrixRow < r.DataShards; matrixRow++ {
		if len(shards[matrixRow]) != 0 {
			subShards[subMatrixRow] = shards[matrixRow]
			validIndices[subMatrixRow] = matrixRow
			subMatrixRow++
		} else {
			invalidIndices = append(invalidIndices, matrixRow)
		}
	}

	subMatrix, _ := newMatrix(r.DataShards, r.DataShards)
	for subMatrixRow, validIndex := range validIndices {
		for c := 0; c < r.DataShards; c++ {
			subMatrix[subMatrixRow][c] = r.m[validIndex][c]
		}
	}
	dataDecodeMatrix, err := r.field.MatrixInvert(subMatrix)
	if err != nil {
		return err
	}
	reconstructData, err := r.field.MatrixMultiply(dataDecodeMatrix, subShards)
	data := reconstructData[0:r.DataShards]
	encoderResult, _ := r.field.MatrixMultiply(r.m, data)
	copy(shards, encoderResult)
	fmt.Println("DecoderResult:", shards)
	return err
}

func (r *Raid6) Split(data []byte) ([][]byte, error) {
	if len(data) == 0 {
		return nil, ErrShortData
	}
	dataLen := len(data)
	// Calculate number of bytes per data shard.
	perShard := (len(data) + r.DataShards - 1) / r.DataShards

	if cap(data) > len(data) {
		data = data[:cap(data)]
	}

	// Only allocate memory if necessary
	var padding []byte
	if len(data) < (r.Shards * perShard) {
		// calculate maximum number of full shards in `data` slice
		fullShards := len(data) / perShard
		padding = make([]byte, r.Shards*perShard-perShard*fullShards)
		copy(padding, data[perShard*fullShards:])
		data = data[0 : perShard*fullShards]
	} else {
		for i := dataLen; i < dataLen+r.DataShards; i++ {
			data[i] = 0
		}
	}

	// Split into equal-length shards.
	dst := make([][]byte, r.Shards)
	i := 0
	for ; i < len(dst) && len(data) >= perShard; i++ {
		dst[i] = data[:perShard:perShard]
		data = data[perShard:]
	}

	for j := 0; i+j < len(dst); j++ {
		dst[i+j] = padding[:perShard:perShard]
		padding = padding[perShard:]
	}

	return dst, nil
}
func (r *Raid6) Join(dst io.Writer, shards [][]byte, outSize int) error {

	if len(shards) < r.DataShards {
		return ErrTooFewShards
	}
	shards = shards[:r.DataShards]
	size := 0
	for _, shard := range shards {
		if shard == nil {
			return ErrReconstructRequired
		}
		size += len(shard)
		if size >= outSize {
			break
		}
	}
	if size < outSize {
		return ErrShortData
	}

	write := outSize
	for _, shard := range shards {
		if write < len(shard) {
			_, err := dst.Write(shard[:write])
			return err
		}
		n, err := dst.Write(shard)
		if err != nil {
			return err
		}
		write -= n
	}
	return nil
}
