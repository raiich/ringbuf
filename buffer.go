package ringbuf

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrOutOfRange     = errors.New("out of range")
	ErrBufferOverflow = errors.New("buffer overflow")
	ErrInvalidState   = errors.New("invalid state")
)

type Position = int32

type Buffer[F any] interface {
	Drop(i Position) error
	Append(item F) error
	Iterator(start Position) (*Iterator[F], error)
	ToSlice(start Position) ([]F, error)
}

func NewRingBuf[F any](size int) *RingBuf[F] {
	return &RingBuf[F]{
		buf:  make([]F, size),
		drop: -1,
		base: Position(-size),
		next: size,
	}
}

type RingBuf[F any] struct {
	drop Position
	buf  []F
	base Position
	next int
}

func (b *RingBuf[F]) Drop(drop Position) error {
	if b.next <= int(drop-b.base) { // b.base + b.next <= drop
		return ErrOutOfRange
	}
	b.drop = drop
	return nil
}

func (b *RingBuf[F]) Append(item F) error {
	size := len(b.buf)
	if size < int(b.base-b.drop)+b.next { // drop + len(buf) < b.base + b.next
		return ErrBufferOverflow
	}
	next := b.next % size
	if next == 0 {
		b.base += Position(size)
	}
	b.buf[next] = item
	b.next = next + 1
	return nil
}

func (b *RingBuf[F]) Iterator(start Position) (*Iterator[F], error) {
	head, tail, err := b.iter(start)
	if err != nil {
		return nil, err
	}
	return NewIterator[F](head, tail), nil
}

func (b *RingBuf[F]) ToSlice(start Position) ([]F, error) {
	head, tail, err := b.iter(start)
	if err != nil {
		return nil, err
	}
	return append(head, tail...), nil
}

func (b *RingBuf[F]) iter(start Position) ([]F, []F, error) {
	if begin := start - b.base; 0 <= begin && begin <= Position(b.next) {
		return b.buf[begin:b.next], nil, nil
	}
	if diff := int(b.base - start); 0 < diff && b.next+diff <= len(b.buf) {
		begin := len(b.buf) - diff
		return b.buf[begin:], b.buf[:b.next], nil
	}
	bottom := b.base - Position(len(b.buf)-b.next)
	upper := b.base + Position(b.next)
	return nil, nil, fmt.Errorf("%w: %v not in range [%v, %v)",
		ErrOutOfRange, start, bottom, upper)
}

func NewSyncBuf[F any](buf Buffer[F]) *SyncBuf[F] {
	return &SyncBuf[F]{
		mu:  sync.RWMutex{},
		buf: buf,
	}
}

type SyncBuf[F any] struct {
	mu  sync.RWMutex
	buf Buffer[F]
}

func (c *SyncBuf[F]) Drop(drop Position) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.Drop(drop)
}

func (c *SyncBuf[F]) Append(item F) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.Append(item)
}

func (c *SyncBuf[F]) ToSlice(start Position) ([]F, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	items, err := c.buf.ToSlice(start)
	if err != nil {
		return nil, err
	}
	ret := make([]F, len(items))
	copy(ret, items)
	return ret, nil
}

func (c *SyncBuf[F]) Iterator(start Position) (*Iterator[F], error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	iter, err := c.buf.Iterator(start)
	if err != nil {
		return nil, err
	}
	ss := make([][]F, len(iter.ss))
	for i, base := range iter.ss {
		ss[i] = make([]F, len(base))
		copy(ss[i], base)
	}
	return NewIterator[F](ss...), nil
}

func NewIterator[F any](slices ...[]F) *Iterator[F] {
	return &Iterator[F]{
		ss:   slices,
		slot: 0,
		idx:  -1,
	}
}

type Iterator[F any] struct {
	ss   [][]F
	slot int
	idx  int
}

func (r *Iterator[F]) Scan() bool {
	r.idx++
	if r.idx < len(r.ss[r.slot]) {
		return true
	}
	r.slot++
	if r.slot < len(r.ss) && len(r.ss[r.slot]) > 0 {
		r.idx = 0
		return true
	}
	return false
}

func (r *Iterator[F]) Item() F {
	return r.ss[r.slot][r.idx]
}

func (r *Iterator[F]) ToSlice() []F {
	var ret []F
	for r.Scan() {
		ret = append(ret, r.Item())
	}
	return ret
}
