package tunnel

import (
	"bytes"
	"testing"
)

func TestMarshalRoundTrip(t *testing.T) {
	want := map[string][]byte{
		"ac": {0x01},
		"id": []byte("tunnel-id"),
		"dt": []byte("binary\x00payload"),
	}

	got, err := Unmarshal(Marshal(want))
	if err != nil {
		t.Fatalf("Unmarshal(Marshal()) error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d fields, want %d", len(got), len(want))
	}
	for key, value := range want {
		if !bytes.Equal(got[key], value) {
			t.Errorf("field %q = %q, want %q", key, got[key], value)
		}
	}
}

func TestUnmarshalRejectsTrailingTruncatedField(t *testing.T) {
	if _, err := Unmarshal([]byte{0}); err == nil {
		t.Fatal("Unmarshal accepted a truncated field")
	}
}

func BenchmarkMarshal(b *testing.B) {
	values := map[string][]byte{
		"ac": {0x01},
		"id": []byte("tunnel-id"),
		"dt": make([]byte, 32*1024),
	}
	b.SetBytes(int64(len(values["dt"])))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Marshal(values)
	}
}
