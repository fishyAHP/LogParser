package segments

import "fmt"

type FileSize uint64

const (
	Byte           FileSize = 1
	KByte                   = 1024 * Byte
	MByte                   = 1024 * KByte
	MaxSegmentSize          = 100 * MByte

	loadFactor = 87.5 / 100
)

func (f FileSize) String() string {
	switch {
	case f >= MByte:
		return fmt.Sprintf("%.2fmb", float64(f)/float64(MByte))
	case f >= KByte:
		return fmt.Sprintf("%.2fkb", float64(f)/float64(KByte))
	default:
		return fmt.Sprintf("%db", f)
	}
}
