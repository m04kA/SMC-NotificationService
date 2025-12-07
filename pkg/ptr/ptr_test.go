package ptr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func FuzzPtr_Int(f *testing.F) {
	f.Add(0)
	f.Add(1)
	f.Add(100)
	f.Add(-1)
	f.Add(-100)
	f.Fuzz(func(t *testing.T, i int) {
		ptr := Ptr(i)
		assert.Equal(t, ptr, &i)
	})
}

func FuzzPtr_Int32(f *testing.F) {
	f.Add(int32(0))
	f.Add(int32(1))
	f.Add(int32(100))
	f.Add(int32(-1))
	f.Add(int32(-100))
	f.Fuzz(func(t *testing.T, i int32) {
		ptr := Ptr(i)
		assert.Equal(t, ptr, &i)
	})
}

func FuzzPtr_Uint32(f *testing.F) {
	f.Add(uint32(0))
	f.Add(uint32(1))
	f.Add(uint32(100))
	f.Fuzz(func(t *testing.T, i uint32) {
		ptr := Ptr(i)
		assert.Equal(t, ptr, &i)
	})
}

func FuzzPtr_Int64(f *testing.F) {
	f.Add(int64(0))
	f.Add(int64(1))
	f.Add(int64(100))
	f.Add(int64(-1))
	f.Add(int64(-100))
	f.Fuzz(func(t *testing.T, i int64) {
		ptr := Ptr(i)
		assert.Equal(t, ptr, &i)
	})
}

func FuzzPtr_Uint64(f *testing.F) {
	f.Add(uint64(0))
	f.Add(uint64(1))
	f.Add(uint64(100))
	f.Fuzz(func(t *testing.T, i uint64) {
		ptr := Ptr(i)
		assert.Equal(t, ptr, &i)
	})
}

func FuzzPtr_Float32(f *testing.F) {
	f.Add(float32(0))
	f.Add(float32(1.0))
	f.Add(float32(100.0))
	f.Fuzz(func(t *testing.T, f32 float32) {
		ptr := Ptr(f32)
		assert.Equal(t, ptr, &f32)
	})
}

func FuzzPtr_Float64(f *testing.F) {
	f.Add(float64(0))
	f.Add(1.0)
	f.Add(100.0)
	f.Add(-1.0)
	f.Add(-100.0)
	f.Fuzz(func(t *testing.T, f64 float64) {
		ptr := Ptr(f64)
		assert.Equal(t, ptr, &f64)
	})
}

func FuzzPtr_String(f *testing.F) {
	f.Add("")
	f.Add("a")
	f.Add("ab")
	f.Add("abc")
	f.Fuzz(func(t *testing.T, s string) {
		ptr := Ptr(s)
		assert.Equal(t, ptr, &s)
	})
}
