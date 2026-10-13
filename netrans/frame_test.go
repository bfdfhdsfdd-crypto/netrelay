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

func BenchmarkMarshalBinaryBase64(b *testing.B) {
	data := make([]byte, 32*1024)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}
	frame := NewDataFrame(data)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = frame.MarshalBinaryBase64()
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

func TestReadFrameBase64LeavesFollowingFrameUnread(t *testing.T) {
	assert := require.New(t)

	first := NewDataFrame([]byte("first frame")).MarshalBinaryBase64()
	second := NewDataFrame([]byte("second frame")).MarshalBinaryBase64()
	reader := bytes.NewReader(append(first, second...))

	decodedFirst, err := ReadFrameBase64(reader)
	assert.NoError(err)
	assert.Equal([]byte("first frame"), decodedFirst.Data)

	decodedSecond, err := ReadFrameBase64(reader)
	assert.NoError(err)
	assert.Equal([]byte("second frame"), decodedSecond.Data)
}

func TestReadFrameRejectsInvalidBase64Length(t *testing.T) {
	assert := require.New(t)

	var header [6]byte
	binary.BigEndian.PutUint32(header[2:], 1)
	encodedHeader := make([]byte, base64.RawURLEncoding.EncodedLen(len(header)))
	base64.RawURLEncoding.Encode(encodedHeader, header[:])

	_, err := ReadFrameBase64(bytes.NewReader(append(encodedHeader, 'A')))
	assert.ErrorContains(err, "invalid base64 frame length")
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

func TestMarshalBinaryBase64PreservesLargeFrameWireFormat(t *testing.T) {
	assert := require.New(t)

	for _, size := range []int{4091, 4092, 4093, 8184, 8191} {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i)
		}
		frame := &DataFrame{Obs: []byte{0x12, 0x34}, Length: uint32(size), Data: data}
		encodedData := make([]byte, base64.RawURLEncoding.EncodedLen(size))
		for i, value := range data {
			encodedData[i] = value ^ frame.Obs[i%2]
		}
		encodedData = []byte(base64.RawURLEncoding.EncodeToString(encodedData[:size]))

		var length [4]byte
		binary.BigEndian.PutUint32(length[:], uint32(len(encodedData)))
		for i := range length {
			length[i] ^= frame.Obs[i%2]
		}
		header := append(append([]byte(nil), frame.Obs...), length[:]...)
		want := append([]byte(base64.RawURLEncoding.EncodeToString(header)), encodedData...)
		assert.Equalf(want, frame.MarshalBinaryBase64(), "payload size %d", size)
	}
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
