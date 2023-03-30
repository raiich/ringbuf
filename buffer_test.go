package ringbuf

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRingBuffer(t *testing.T) {
	buf := NewRingBuf[Item](3)
	item := Item(nil)

	var items []Item
	var err error
	items, err = buf.ToSlice(0)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(items))

	assert.NoError(t, buf.Append(item)) // 0
	assert.NoError(t, buf.Append(item)) // 1
	assert.NoError(t, buf.Append(item)) // 2
	assert.Equal(t, ErrBufferOverflow, buf.Append(item))

	items, err = buf.ToSlice(0)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(items))

	assert.NoError(t, buf.Drop(0))
	assert.NoError(t, buf.Append(item)) // 3
	items, err = buf.ToSlice(1)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(items))

	assert.NoError(t, buf.Drop(2))
	assert.NoError(t, buf.Append(item)) // 4
	assert.NoError(t, buf.Append(item)) // 5

	items, err = buf.ToSlice(3)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(items))

	assert.Equal(t, ErrBufferOverflow, buf.Append(item))

	assert.Equal(t, ErrOutOfRange, buf.Drop(6))
}

func TestRingBufferIterate(t *testing.T) {
	buf := &RingBuf[Item]{
		buf:  make([]Item, 3),
		drop: 0,
		base: 3,
		next: 1,
	}
	var items []Item
	var err error

	_, err = buf.ToSlice(0)
	assert.True(t, errors.Is(err, ErrOutOfRange))

	items, err = buf.ToSlice(1)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(items))

	items, err = buf.ToSlice(2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(items))

	items, err = buf.ToSlice(3)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(items))

	items, err = buf.ToSlice(4)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(items))

	_, err = buf.ToSlice(5)
	assert.True(t, errors.Is(err, ErrOutOfRange))
}

func TestRingBufferIterator(t *testing.T) {
	buf := &RingBuf[Item]{
		buf:  make([]Item, 3),
		drop: 0,
		base: 3,
		next: 1,
	}
	//var items Iterator[Item]
	var err error

	_, err = buf.Iterator(0)
	assert.True(t, errors.Is(err, ErrOutOfRange))

	items, err := buf.Iterator(1)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(items.ToSlice()))

	items, err = buf.Iterator(2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(items.ToSlice()))

	items, err = buf.Iterator(3)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(items.ToSlice()))

	items, err = buf.Iterator(4)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(items.ToSlice()))

	_, err = buf.Iterator(5)
	assert.True(t, errors.Is(err, ErrOutOfRange))
}

func TestRingBufferAroundZero(t *testing.T) {
	buf := &RingBuf[Item]{
		buf:  make([]Item, 3),
		drop: -6,
		base: -4,
		next: 2,
	}
	checkAppendAndIterate(t, buf, -4) // -4, -3, -2
	checkAppendAndIterate(t, buf, -3) // -3, -2, -1
	checkAppendAndIterate(t, buf, -2) // -2, -1, 0
	checkAppendAndIterate(t, buf, -1) // -1, 0, 1
	checkAppendAndIterate(t, buf, 0)  // 0, 1, 2
	checkAppendAndIterate(t, buf, 1)  // 1, 2, 3
}

func TestRingBufferWrapAround(t *testing.T) {
	var large int32 = (1 << 31) - 1
	buf := &RingBuf[Item]{
		buf:  make([]Item, 3),
		drop: large - 3,
		base: large,
		next: 1,
	}
	checkAppendAndIterate(t, buf, large-1) // large-1, large, large+1
	checkAppendAndIterate(t, buf, large)   // large, large+1, large+2
	checkAppendAndIterate(t, buf, large+1) // large+1, large+2, large+3
	checkAppendAndIterate(t, buf, large+2) // large+2, large+3, large+4
	checkAppendAndIterate(t, buf, large+3) // large+3, large+4, large+5
	checkAppendAndIterate(t, buf, large+4) // large+4, large+5, large+6
}

func checkAppendAndIterate(t *testing.T, buf *RingBuf[Item], start Position) {
	t.Helper()
	{
		item := Item(nil)
		assert.True(t, errors.Is(buf.Append(item), ErrBufferOverflow))
		assert.NoError(t, buf.Drop(start-1))
		assert.NoError(t, buf.Append(item))
	}
	{
		var items []Item
		var err error
		_, err = buf.ToSlice(start - 1)
		assert.True(t, errors.Is(err, ErrOutOfRange))

		items, err = buf.ToSlice(start)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(items))

		items, err = buf.ToSlice(start + 1)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(items))

		items, err = buf.ToSlice(start + 2)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(items))

		items, err = buf.ToSlice(start + 3)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(items))

		_, err = buf.ToSlice(start + 4)
		assert.True(t, errors.Is(err, ErrOutOfRange))
	}
}

type Item = any
