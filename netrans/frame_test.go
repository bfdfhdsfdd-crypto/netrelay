package netrans

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestFrame(t *testing.T) {
	assert := require.New(t)

	for i := 0; i < 1000; i++ {
		data := make([]byte, 1000*i%10000)
		_, err := rand.Read(data)
		assert.Nil(err)

		fr := NewDataFrame(data)
		bin := fr.MarshalBinary()

		newFr, err := ReadFrame(bytes.NewReader(bin))
		assert.Nil(err)
		assert.Equal(newFr.Length, uint32(len(data)))
		assert.Equal(data, newFr.Data)
	}
}

func TestFrameBase64(t *testing.T) {
	assert := require.New(t)

	for i := 0; i < 1000; i++ {
		data := make([]byte, 1000*i)
		_, err := rand.Read(data)
		assert.Nil(err)

		fr := NewDataFrame(data)
		bin := fr.MarshalBinaryBase64()

		newFr, err := ReadFrameBase64(bytes.NewReader(bin))
		assert.Nil(err)
		assert.Equal(newFr.Length, uint32(len(data)))
		assert.Equal(data, newFr.Data)
	}
}

func TestReadFrameRejectsOversizedBase64Frame(t *testing.T) {
	assert := require.New(t)

	maxEncodedLength := base64.RawURLEncoding.EncodedLen(maxFrameSize)
	var rawHeader [6]byte
	binary.BigEndian.PutUint32(rawHeader[2:], uint32(maxEncodedLength+1))
	header := make([]byte, base64.RawURLEncoding.EncodedLen(len(rawHeader)))
	base64.RawURLEncoding.Encode(header, rawHeader[:])

	_, err := ReadFrameBase64(bytes.NewReader(header))
	assert.ErrorContains(err, "encoded frame is too big")
}

func TestMarshalBinaryBase64PreservesWireFormat(t *testing.T) {
	assert := require.New(t)

	data := []byte("wire compatibility")
	frame := &DataFrame{Obs: []byte{0x12, 0x34}, Length: uint32(len(data)), Data: data}
	encodedData := append([]byte(nil), data...)
	for i := range encodedData {
		encodedData[i] ^= frame.Obs[i%2]
	}
	encodedData = []byte(base64.RawURLEncoding.EncodeToString(encodedData))

	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(encodedData)))
	for i := range length {
		length[i] ^= frame.Obs[i%2]
	}
	header := append(append([]byte(nil), frame.Obs...), length[:]...)
	want := append([]byte(base64.RawURLEncoding.EncodeToString(header)), encodedData...)

	assert.Equal(want, frame.MarshalBinaryBase64())
}

func TestReadFrameReadsPartialObfuscationHeader(t *testing.T) {
	assert := require.New(t)

	data := []byte("frame payload")
	frame := NewDataFrame(data).MarshalBinary()
	decoded, err := ReadFrame(&oneByteReader{r: bytes.NewReader(frame)})
	assert.NoError(err)
	assert.Equal(data, decoded.Data)
}

type oneByteReader struct {
	r io.Reader
}

func (r *oneByteReader) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return r.r.Read(p)
}
