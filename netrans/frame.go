package netrans

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const maxFrameSize = 32 * 1024 * 1024
const frameEncodeChunkSize = 4092

type DataFrame struct {
	Obs    []byte
	Length uint32
	Data   []byte
}

func NewDataFrame(data []byte) *DataFrame {
	obs := make([]byte, 2)
	_, _ = rand.Read(obs[:])
	return &DataFrame{
		Obs:    obs,
		Length: uint32(len(data)),
		Data:   data,
	}
}

func (d *DataFrame) MarshalBinaryBase64() []byte {
	encodedDataLength := base64.RawURLEncoding.EncodedLen(len(d.Data))
	var header [6]byte
	copy(header[:2], d.Obs)
	binary.BigEndian.PutUint32(header[2:], uint32(encodedDataLength))
	for i := 2; i < len(header); i++ {
		header[i] ^= d.Obs[(i-2)%2]
	}

	result := make([]byte, 8+encodedDataLength)
	base64.RawURLEncoding.Encode(result[:8], header[:])

	var transformed [frameEncodeChunkSize]byte
	encodedOffset := 0
	for offset := 0; offset < len(d.Data); {
		chunkSize := min(frameEncodeChunkSize, len(d.Data)-offset)
		chunk := transformed[:chunkSize]
		copy(chunk, d.Data[offset:offset+chunkSize])
		for i := range chunk {
			chunk[i] ^= d.Obs[(offset+i)%2]
		}
		chunkEncodedLength := base64.RawURLEncoding.EncodedLen(chunkSize)
		base64.RawURLEncoding.Encode(
			result[8+encodedOffset:8+encodedOffset+chunkEncodedLength],
			chunk,
		)
		offset += chunkSize
		encodedOffset += chunkEncodedLength
	}
	return result
}

func (d *DataFrame) MarshalBinary() []byte {
	result := make([]byte, 4, 4+2+d.Length)
	binary.BigEndian.PutUint32(result, d.Length)
	result = append(result, d.Obs...)
	result = append(result, d.Data...)
	for i := 6; i < len(result); i++ {
		result[i] = result[i] ^ d.Obs[(i-6)%2]
	}
	return result
}

func ReadFrame(r io.Reader) (*DataFrame, error) {
	var bs [4]byte
	// read xor and magic number
	_, err := io.ReadFull(r, bs[:])
	if err != nil {
		return nil, err
	}
	fr := &DataFrame{}

	fr.Length = binary.BigEndian.Uint32(bs[:])
	if fr.Length > maxFrameSize {
		return nil, fmt.Errorf("frame is too big, %d", fr.Length)
	}
	var obs [2]byte
	_, err = io.ReadFull(r, obs[:])
	if err != nil {
		return nil, fmt.Errorf("read type error %v", err)
	}
	fr.Obs = obs[:]
	buf := make([]byte, fr.Length)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, fmt.Errorf("read data error: %v", err)
	}
	for i := 0; i < len(buf); i++ {
		buf[i] = buf[i] ^ fr.Obs[i%2]
	}
	fr.Data = buf
	return fr, nil
}

// ReadFrameBase64 reads a base64 encoded DataFrame from an io.Reader.
func ReadFrameBase64(r io.Reader) (*DataFrame, error) {
	// Read the first part (length and obs)
	// The first part is base64 encoded from 6 bytes (4 bytes length + 2 bytes obs).
	// Base64 encoding 6 bytes results in 8 bytes.
	var headerBase64 [8]byte
	n, err := io.ReadFull(r, headerBase64[:])
	if err != nil {
		return nil, errors.New("failed to read header base64: " + err.Error())
	}
	if n != 8 {
		return nil, errors.New("incomplete header base64 read")
	}
	var headerBytes [6]byte
	n, err = base64.RawURLEncoding.Decode(headerBytes[:], headerBase64[:])
	if err != nil {
		return nil, errors.New("failed to decode header base64: " + err.Error())
	}
	if n != len(headerBytes) {
		return nil, errors.New("invalid header length, expected 6 bytes after decoding")
	}
	// Extract obs and length from the decoded header
	obs := headerBytes[:2]
	for i := 2; i < 6; i++ {
		headerBytes[i] = headerBytes[i] ^ obs[(i-2)%2]
	}
	dataLength := binary.BigEndian.Uint32(headerBytes[2:])
	maxEncodedLength := base64.RawURLEncoding.EncodedLen(maxFrameSize)
	if uint64(dataLength) > uint64(maxEncodedLength) {
		return nil, fmt.Errorf("encoded frame is too big, %d", dataLength)
	}
	if dataLength%4 == 1 {
		return nil, errors.New("invalid base64 frame length")
	}

	dataBytes := make([]byte, base64.RawURLEncoding.DecodedLen(int(dataLength)))
	_, err = io.ReadFull(base64.NewDecoder(base64.RawURLEncoding, io.LimitReader(r, int64(dataLength))), dataBytes)
	if err != nil {
		return nil, errors.New("failed to read or decode data base64: " + err.Error())
	}
	// Decode the data using obs
	for i := 0; i < len(dataBytes); i++ {
		dataBytes[i] = dataBytes[i] ^ obs[i%2]
	}
	return &DataFrame{
		Length: uint32(len(dataBytes)),
		Obs:    obs,
		Data:   dataBytes,
	}, nil
}
