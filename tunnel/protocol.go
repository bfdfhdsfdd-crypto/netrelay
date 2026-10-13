package tunnel

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"strconv"

	"github.com/bfdfhdsfdd-crypto/netrelay/netrans"
)

type ConnectionType string

const (
	Checking   ConnectionType = "checking"
	AutoDuplex ConnectionType = "auto"
	FullDuplex ConnectionType = "full"
	HalfDuplex ConnectionType = "half"
	Classic    ConnectionType = "classic"
)

func AllConnectionTypes() []ConnectionType {
	return []ConnectionType{
		Checking,
		AutoDuplex,
		FullDuplex,
		HalfDuplex,
		Classic,
	}
}

func (c ConnectionType) Bin() byte {
	switch c {
	case Checking:
		return 0x00
	case FullDuplex:
		return 0x01
	case HalfDuplex:
		return 0x02
	case Classic:
		return 0x03
	default:
		return 0xff
	}
}

const (
	ActionCreate    byte = 0x00
	ActionData      byte = 0x01
	ActionDelete    byte = 0x02
	ActionStatus    byte = 0x03
	ActionHeartbeat byte = 0x10
	ActionDirty     byte = 0x11
)

func BuildBody(m map[string][]byte, redirect, sid string, ct ConnectionType) []byte {
	if len(redirect) != 0 {
		m["r"] = []byte(redirect)
	}
	if len(sid) != 0 {
		m["sid"] = []byte(sid)
	}
	m["m"] = []byte{ct.Bin()}
	// some junk data
	m["_"] = RandBytes(64)
	return netrans.NewDataFrame(Marshal(m)).MarshalBinaryBase64()
}

func NewActionCreate(id, addr string, port uint16) map[string][]byte {
	m := make(map[string][]byte)
	m["ac"] = []byte{ActionCreate}
	m["id"] = []byte(id)
	m["h"] = []byte(addr)
	m["p"] = []byte(strconv.Itoa(int(port)))

	return m
}

func NewActionData(id string, data []byte) map[string][]byte {
	m := make(map[string][]byte)
	m["ac"] = []byte{ActionData}
	m["id"] = []byte(id)
	m["dt"] = []byte(data)
	return m
}

func NewActionDelete(id string) map[string][]byte {
	m := make(map[string][]byte)
	m["ac"] = []byte{ActionDelete}
	m["id"] = []byte(id)
	return m
}

func NewActionHeartbeat(id string) map[string][]byte {
	m := make(map[string][]byte)
	m["ac"] = []byte{ActionHeartbeat}
	m["id"] = []byte(id)
	return m
}

// 定义一个最简的序列化协议，k,v 交替，每一项是len+data
// 其中 k 最长 255，v 最长 MaxUInt32
func Marshal(m map[string][]byte) []byte {
	size := 0
	for k, v := range m {
		size += 1 + len(k) + 4 + len(v)
	}

	buf := make([]byte, size)
	offset := 0
	for k, v := range m {
		buf[offset] = byte(len(k))
		offset++
		offset += copy(buf[offset:], k)
		binary.BigEndian.PutUint32(buf[offset:], uint32(len(v)))
		offset += 4
		offset += copy(buf[offset:], v)
	}
	return buf
}

func Unmarshal(bs []byte) (map[string][]byte, error) {
	m := make(map[string][]byte)
	total := len(bs)
	for i := 0; i < total; {
		kLen := int(bs[i])
		i += 1

		if i+kLen > total {
			return nil, fmt.Errorf("unexpected eof when read key")
		}
		key := string(bs[i : i+kLen])
		i += kLen

		if i+4 > total {
			return nil, fmt.Errorf("unexpected eof when read value size")
		}
		valueLength := binary.BigEndian.Uint32(bs[i : i+4])
		i += 4

		if uint64(valueLength) > uint64(total-i) {
			return nil, fmt.Errorf("unexpected eof when read value")
		}
		vLen := int(valueLength)
		value := bs[i : i+vLen]
		m[key] = value
		i += vLen
	}
	return m, nil
}

func UnmarshalFrameWithBuffer(r io.Reader) (map[string][]byte, []byte, error) {
	var buf bytes.Buffer
	teeReader := io.TeeReader(r, &buf)
	fr, err := netrans.ReadFrameBase64(teeReader)
	if err != nil {
		return nil, buf.Bytes(), err
	}

	serverData, err := Unmarshal(fr.Data)
	if err != nil {
		return nil, nil, err
	}
	return serverData, buf.Bytes(), nil
}

func RandBytes(max int) []byte {
	length := rand.Intn(max)
	b := make([]byte, length)
	rand.Read(b) //nolint:staticcheck
	return b
}
