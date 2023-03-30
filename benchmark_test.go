package ringbuf

import (
	"testing"
)

var (
	size = 1024 * 32
	skip = 4
)

func BenchmarkBufAppend(b *testing.B) {
	cases := []struct {
		name string
		buf  func() Buffer[int]
	}{
		{name: "ring", buf: func() Buffer[int] { return NewRingBuf[int](size) }},
		{name: "slice", buf: func() Buffer[int] { return NewSliceBuf[int](size) }},
		{name: "ring-sync", buf: func() Buffer[int] { return NewSyncBuf[int](NewRingBuf[int](size)) }},
		{name: "slice-sync", buf: func() Buffer[int] { return NewSyncBuf[int](NewSliceBuf[int](size)) }},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			buf := c.buf()
			for i := 0; i < b.N; i++ {
				if i >= size-skip && i%skip == 0 {
					if err := buf.Drop(Position(i - size + skip)); err != nil {
						panic(err)
					}
				}
				if err := buf.Append(i); err != nil {
					panic(err)
				}
			}
		})
	}
}

func BenchmarkBufToSlice(b *testing.B) {
	cases := []struct {
		name string
		buf  func() Buffer[int]
	}{
		{name: "ring", buf: func() Buffer[int] { return NewRingBuf[int](size) }},
		{name: "slice", buf: func() Buffer[int] { return NewSliceBuf[int](size) }},
		{name: "ring-sync", buf: func() Buffer[int] { return NewSyncBuf[int](NewRingBuf[int](size)) }},
		{name: "slice-sync", buf: func() Buffer[int] { return NewSyncBuf[int](NewSliceBuf[int](size)) }},
	}
	max := (size * 3) / 2
	start := max - size
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			buf := c.buf()
			for i := 0; i < max; i++ {
				if i >= size {
					if err := buf.Drop(Position(i - size)); err != nil {
						panic(err)
					}
				}
				if err := buf.Append(i); err != nil {
					panic(err)
				}
			}
			for i := 0; i < b.N; i++ {
				_, err := buf.ToSlice(Position(start))
				if err != nil {
					panic(err)
				}
			}
		})
	}
}

func BenchmarkBufIterator(b *testing.B) {
	cases := []struct {
		name string
		buf  func() Buffer[int]
	}{
		{name: "ring", buf: func() Buffer[int] { return NewRingBuf[int](size) }},
		{name: "slice", buf: func() Buffer[int] { return NewSliceBuf[int](size) }},
		{name: "ring-sync", buf: func() Buffer[int] { return NewSyncBuf[int](NewRingBuf[int](size)) }},
		{name: "slice-sync", buf: func() Buffer[int] { return NewSyncBuf[int](NewSliceBuf[int](size)) }},
	}
	max := (size * 3) / 2
	start := max - size
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			buf := c.buf()
			for i := 0; i < max; i++ {
				if i >= size {
					if err := buf.Drop(Position(i - size)); err != nil {
						panic(err)
					}
				}
				if err := buf.Append(i); err != nil {
					panic(err)
				}
			}
			for i := 0; i < b.N; i++ {
				_, err := buf.Iterator(Position(start))
				if err != nil {
					panic(err)
				}
			}
		})
	}
}

func NewSliceBuf[F any](size int) *SliceBuf[F] {
	return &SliceBuf[F]{
		size: size,
		buf:  nil,
		base: 0,
	}
}

type SliceBuf[F any] struct {
	size int
	buf  []F
	base Position
}

func (b *SliceBuf[F]) Drop(drop Position) error {
	base := b.base
	if len(b.buf) <= int(drop-base) { // base + len(buf) <= drop
		return ErrOutOfRange
	}
	if 0 <= drop-base { // base <= drop
		b.buf = b.buf[drop-base+1:]
		b.base = drop + 1
	}
	return nil
}

func (b *SliceBuf[F]) Append(item F) error {
	if b.size <= len(b.buf) {
		return ErrBufferOverflow
	}
	b.buf = append(b.buf, item)
	return nil
}

func (b *SliceBuf[F]) Iterator(start Position) (*Iterator[F], error) {
	ss, err := b.ToSlice(start)
	if err != nil {
		return nil, err
	}
	return NewIterator[F](ss), nil
}

func (b *SliceBuf[F]) ToSlice(start Position) ([]F, error) {
	if start-b.base < 0 || len(b.buf) < int(start-b.base) {
		return nil, ErrOutOfRange
	}
	return b.buf[start-b.base:], nil
}
