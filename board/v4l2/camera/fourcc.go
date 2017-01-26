package camera

import (
	"bytes"
	"fmt"
)

// FourCC is V4L2 fourcc format
type FourCC uint32

// FourCC formats
const (
	FourCCMJPG FourCC = 0x47504a4d
)

// ParseFourCC parses FourCC from a string
func ParseFourCC(str string) (FourCC, error) {
	if len(str) != 4 {
		return FourCC(0), fmt.Errorf("incorrect length %d of FourCC, must be 4", len(str))
	}
	var v uint32
	for i := 0; i < 4; i++ {
		v |= uint32(uint8(str[i])) << uint(i*8)
	}
	return FourCC(v), nil
}

// String returns the string represent of fourcc
func (cc FourCC) String() (s string) {
	for i := uint(0); i < 4; i++ {
		s += string(rune((cc >> (i * 8)) & 0xff))
	}
	return
}

// MarshalJSON implements json.Marshaler
func (cc FourCC) MarshalJSON() ([]byte, error) {
	return []byte(`"` + cc.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler
func (cc *FourCC) UnmarshalJSON(data []byte) error {
	if !bytes.HasPrefix(data, []byte{'"'}) ||
		!bytes.HasSuffix(data, []byte{'"'}) {
		return fmt.Errorf("not a string %s", string(data))
	}
	if len(data) != 6 {
		return fmt.Errorf("incorrect length %d", len(data))
	}
	parsed, err := ParseFourCC(string(data[1:5]))
	if err == nil {
		*cc = parsed
	}
	return err
}
