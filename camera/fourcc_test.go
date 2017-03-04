package camera

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testFourCCStruct struct {
	CC FourCC `json:"fourcc"`
}

func TestFourCCEncodeDecode(t *testing.T) {
	var cc FourCC
	assert.NoError(t, json.Unmarshal([]byte(`"MJPG"`), &cc))
	assert.Equal(t, FourCCMJPG, cc)
	data, err := json.Marshal(cc)
	assert.NoError(t, err)
	assert.Equal(t, `"MJPG"`, string(data))
}

func TestFourCCInStruct(t *testing.T) {
	var stru testFourCCStruct
	assert.NoError(t, json.Unmarshal([]byte(`{"fourcc":"MJPG"}`), &stru))
	assert.Equal(t, FourCCMJPG, stru.CC)
	data, err := json.Marshal(stru)
	assert.NoError(t, err)
	assert.Equal(t, `{"fourcc":"MJPG"}`, string(data))
}
