package erasure

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Shard represents an individual data or parity chunk shard with cryptographic checksum.
type Shard struct {
	Index    int    `json:"index"`
	Data     []byte `json:"data"`
	Checksum string `json:"checksum"` // SHA-256 hex checksum (finding #17: tamper detection)
	IsParity bool   `json:"is_parity"`
}

// ComputeChecksum calculates the SHA-256 hex checksum of shard data.
func ComputeChecksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Encoder provides Reed-Solomon $K+M$ erasure coding operations.
type Encoder struct {
	K int // Data shards count
	M int // Parity shards count
}

func NewEncoder(k, m int) (*Encoder, error) {
	if k <= 0 || m <= 0 {
		return nil, fmt.Errorf("invalid shard counts: K=%d, M=%d (must be > 0)", k, m)
	}
	return &Encoder{K: k, M: m}, nil
}

// Encode splits data into K data shards and generates M parity shards using Galois Field GF(2^8) operations.
func (e *Encoder) Encode(data []byte) ([]Shard, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("cannot encode empty data")
	}

	shardSize := (len(data) + e.K - 1) / e.K
	paddedSize := shardSize * e.K
	padded := make([]byte, paddedSize)
	copy(padded, data)

	shards := make([]Shard, e.K+e.M)

	// 1. Data Shards
	for i := 0; i < e.K; i++ {
		start := i * shardSize
		end := start + shardSize
		shardData := make([]byte, shardSize)
		copy(shardData, padded[start:end])

		shards[i] = Shard{
			Index:    i,
			Data:     shardData,
			Checksum: ComputeChecksum(shardData),
			IsParity: false,
		}
	}

	// 2. Parity Shards (XOR & Vandermonde GF(2^8) linear combination)
	for p := 0; p < e.M; p++ {
		parityData := make([]byte, shardSize)
		for i := 0; i < e.K; i++ {
			coeff := gfPow(byte(p+1), byte(i))
			for byteIdx := 0; byteIdx < shardSize; byteIdx++ {
				parityData[byteIdx] ^= gfMul(shards[i].Data[byteIdx], coeff)
			}
		}
		shards[e.K+p] = Shard{
			Index:    e.K + p,
			Data:     parityData,
			Checksum: ComputeChecksum(parityData),
			IsParity: true,
		}
	}

	return shards, nil
}


// Reconstruct recovers the original byte data from any K surviving shards.
func (e *Encoder) Reconstruct(available map[int][]byte, originalLength int) ([]byte, error) {
	if len(available) < e.K {
		return nil, fmt.Errorf("insufficient shards for reconstruction: got %d, needed at least K=%d", len(available), e.K)
	}

	// Determine shard size
	var shardSize int
	for _, data := range available {
		shardSize = len(data)
		break
	}

	// Reconstruct all data shards 0..K-1
	recoveredDataShards := make(map[int][]byte)
	for i := 0; i < e.K; i++ {
		if data, ok := available[i]; ok {
			recoveredDataShards[i] = append([]byte(nil), data...)
		}
	}

	// If all data shards are already present, rejoin directly
	if len(recoveredDataShards) == e.K {
		return rejoin(recoveredDataShards, e.K, originalLength), nil
	}

	// Matrix elimination over GF(2^8) to recover missing data shards
	// Build system of linear equations from available shards
	var selectedIndices []int
	for idx := range available {
		if len(selectedIndices) < e.K {
			selectedIndices = append(selectedIndices, idx)
		}
	}

	// Construct encoding matrix rows for selected indices
	matrix := make([][]byte, e.K)
	for rowIdx, shardIdx := range selectedIndices {
		matrix[rowIdx] = make([]byte, e.K)
		if shardIdx < e.K {
			matrix[rowIdx][shardIdx] = 1
		} else {
			p := shardIdx - e.K
			for col := 0; col < e.K; col++ {
				matrix[rowIdx][col] = gfPow(byte(p+1), byte(col))
			}
		}
	}

	// Invert matrix over GF(2^8)
	invMatrix, err := gfInvertMatrix(matrix, e.K)
	if err != nil {
		return nil, fmt.Errorf("failed to invert reconstruction matrix: %w", err)
	}

	// Multiply inverted matrix by available shard column vector to restore missing data shards
	for dataIdx := 0; dataIdx < e.K; dataIdx++ {
		if _, present := recoveredDataShards[dataIdx]; !present {
			recovered := make([]byte, shardSize)
			for colIdx, shardIdx := range selectedIndices {
				coeff := invMatrix[dataIdx][colIdx]
				src := available[shardIdx]
				for b := 0; b < shardSize; b++ {
					recovered[b] ^= gfMul(src[b], coeff)
				}
			}
			recoveredDataShards[dataIdx] = recovered
		}
	}

	return rejoin(recoveredDataShards, e.K, originalLength), nil
}

// ReconstructVerified validates shard checksums before reconstruction, discarding tampered shards (finding #17).
func (e *Encoder) ReconstructVerified(available map[int][]byte, expectedChecksums map[int]string, originalLength int) ([]byte, error) {
	validShards := make(map[int][]byte)
	for idx, data := range available {
		if expected, ok := expectedChecksums[idx]; ok && expected != "" {
			if ComputeChecksum(data) != expected {
				continue // Silently discard tampered/corrupted shard
			}
		}
		validShards[idx] = data
	}

	if len(validShards) < e.K {
		return nil, fmt.Errorf("insufficient valid shards after integrity check: %d valid (needed K=%d)", len(validShards), e.K)
	}

	return e.Reconstruct(validShards, originalLength)
}


func rejoin(shards map[int][]byte, k int, origLen int) []byte {
	var buf []byte
	for i := 0; i < k; i++ {
		buf = append(buf, shards[i]...)
	}
	if origLen > 0 && origLen < len(buf) {
		buf = buf[:origLen]
	}
	return buf
}

// Galois Field GF(2^8) math with primitive polynomial 0x11d
var (
	gfExp [512]byte
	gfLog [256]byte
)

func init() {
	x := 1
	for i := 0; i < 255; i++ {
		gfExp[i] = byte(x)
		gfExp[i+255] = byte(x)
		gfLog[x] = byte(i)
		x <<= 1
		if x&0x100 != 0 {
			x ^= 0x11d
		}
	}
}

func gfMul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return gfExp[int(gfLog[a])+int(gfLog[b])]
}

func gfDiv(a, b byte) byte {
	if a == 0 {
		return 0
	}
	if b == 0 {
		panic("divide by zero in GF(2^8)")
	}
	return gfExp[int(gfLog[a])-int(gfLog[b])+255]
}

func gfPow(a, b byte) byte {
	if b == 0 {
		return 1
	}
	if a == 0 {
		return 0
	}
	res := byte(1)
	for i := 0; i < int(b); i++ {
		res = gfMul(res, a)
	}
	return res
}

func gfInvertMatrix(mat [][]byte, n int) ([][]byte, error) {
	// Create augmented matrix [mat | I]
	aug := make([][]byte, n)
	for i := 0; i < n; i++ {
		aug[i] = make([]byte, 2*n)
		copy(aug[i][:n], mat[i])
		aug[i][n+i] = 1
	}

	// Gaussian elimination over GF(2^8)
	for i := 0; i < n; i++ {
		// Find pivot
		pivot := i
		for j := i; j < n; j++ {
			if aug[j][i] != 0 {
				pivot = j
				break
			}
		}
		if aug[pivot][i] == 0 {
			return nil, fmt.Errorf("singular matrix")
		}
		aug[i], aug[pivot] = aug[pivot], aug[i]

		// Scale pivot row
		inv := gfDiv(1, aug[i][i])
		for j := 0; j < 2*n; j++ {
			aug[i][j] = gfMul(aug[i][j], inv)
		}

		// Eliminate column entries
		for r := 0; r < n; r++ {
			if r != i && aug[r][i] != 0 {
				factor := aug[r][i]
				for c := 0; c < 2*n; c++ {
					aug[r][c] ^= gfMul(aug[i][c], factor)
				}
			}
		}
	}

	// Extract inverse matrix
	inv := make([][]byte, n)
	for i := 0; i < n; i++ {
		inv[i] = make([]byte, n)
		copy(inv[i], aug[i][n:])
	}
	return inv, nil
}
