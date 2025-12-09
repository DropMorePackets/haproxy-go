package encoding

import (
	"net/netip"
	"sync"
	"testing"
)

// TestStruct represents a typical struct that would be unmarshaled
type TestStruct struct {
	Headers  []byte     `spoe:"headers"`
	Status   int32      `spoe:"status-code"`
	ClientIP netip.Addr `spoe:"client-ip"`
	UserID   uint64     `spoe:"user-id"`
	Active   bool       `spoe:"active"`
	Optional *string    `spoe:"optional-field"`
}

var testStructPool = sync.Pool{
	New: func() any {
		return &TestStruct{}
	},
}

func acquireTestStruct() *TestStruct {
	return testStructPool.Get().(*TestStruct)
}

func releaseTestStruct(s *TestStruct) {
	// Reset all fields
	s.Headers = nil
	s.Status = 0
	s.ClientIP = netip.Addr{}
	s.UserID = 0
	s.Active = false
	s.Optional = nil
	testStructPool.Put(s)
}

// setupTestData creates a KV buffer with test data
func setupTestData() []byte {
	buf := make([]byte, 1024)
	w := NewKVWriter(buf, 0)
	w.SetString("headers", "Content-Type: application/json")
	w.SetInt32("status-code", 200)
	addr := netip.MustParseAddr("192.168.1.100")
	w.SetAddr("client-ip", addr)
	w.SetUInt64("user-id", 12345)
	w.SetBool("active", true)
	w.SetString("optional-field", "optional-value")
	return w.Bytes()
}

func BenchmarkUnmarshal(b *testing.B) {
	data := setupTestData()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			scanner := NewKVScanner(data, -1)
			s := acquireTestStruct()
			if err := scanner.Unmarshal(s); err != nil {
				b.Fatal(err)
			}
			releaseTestStruct(s)
			ReleaseKVScanner(scanner)
		}
	})
}

func BenchmarkManualIteration(b *testing.B) {
	data := setupTestData()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			scanner := NewKVScanner(data, -1)
			s := acquireTestStruct()

			entry := AcquireKVEntry()
			for scanner.Next(entry) {
				switch {
				case entry.NameEquals("headers"):
					val := entry.ValueBytes()
					s.Headers = make([]byte, len(val))
					copy(s.Headers, val)
				case entry.NameEquals("status-code"):
					s.Status = int32(entry.ValueInt())
				case entry.NameEquals("client-ip"):
					s.ClientIP = entry.ValueAddr()
				case entry.NameEquals("user-id"):
					s.UserID = uint64(entry.ValueInt())
				case entry.NameEquals("active"):
					s.Active = entry.ValueBool()
				case entry.NameEquals("optional-field"):
					val := entry.Value().(string)
					s.Optional = &val
				}
			}

			if err := scanner.Error(); err != nil {
				b.Fatal(err)
			}

			ReleaseKVEntry(entry)
			releaseTestStruct(s)
			ReleaseKVScanner(scanner)
		}
	})
}

// BenchmarkUnmarshalSequential runs unmarshal sequentially (no parallel)
func BenchmarkUnmarshalSequential(b *testing.B) {
	data := setupTestData()
	s := acquireTestStruct()
	defer releaseTestStruct(s)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner := NewKVScanner(data, -1)
		if err := scanner.Unmarshal(s); err != nil {
			b.Fatal(err)
		}
		ReleaseKVScanner(scanner)
		// Reset struct manually for next iteration
		s.Headers = nil
		s.Status = 0
		s.ClientIP = netip.Addr{}
		s.UserID = 0
		s.Active = false
		s.Optional = nil
	}
}

// BenchmarkManualIterationSequential runs manual iteration sequentially
func BenchmarkManualIterationSequential(b *testing.B) {
	data := setupTestData()
	s := acquireTestStruct()
	defer releaseTestStruct(s)
	entry := AcquireKVEntry()
	defer ReleaseKVEntry(entry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner := NewKVScanner(data, -1)

		for scanner.Next(entry) {
			switch {
			case entry.NameEquals("headers"):
				val := entry.ValueBytes()
				s.Headers = make([]byte, len(val))
				copy(s.Headers, val)
			case entry.NameEquals("status-code"):
				s.Status = int32(entry.ValueInt())
			case entry.NameEquals("client-ip"):
				s.ClientIP = entry.ValueAddr()
			case entry.NameEquals("user-id"):
				s.UserID = uint64(entry.ValueInt())
			case entry.NameEquals("active"):
				s.Active = entry.ValueBool()
			case entry.NameEquals("optional-field"):
				val := entry.Value().(string)
				s.Optional = &val
			}
		}

		if err := scanner.Error(); err != nil {
			b.Fatal(err)
		}

		ReleaseKVScanner(scanner)
		// Reset struct manually for next iteration
		s.Headers = nil
		s.Status = 0
		s.ClientIP = netip.Addr{}
		s.UserID = 0
		s.Active = false
		s.Optional = nil
	}
}
