package circular

import (
	"fmt"
	"sync"
)

/*
 * Data structure implementing a circular buffer.
 */
type Buffer[T any] struct {
	mutex   sync.RWMutex
	values  []T
	pointer int
}

/*
 * Add elements to the circular buffer, potentially overwriting unread elements.
 *
 * Semantics: First write to buffer, then increment pointer.
 *
 * Pointer points to "oldest" element, or next element to be overwritten.
 */
func (b *Buffer[T]) Enqueue(elems ...T) {
	numElems := len(elems)
	values := b.values
	n := len(values)

	/*
	 * If there are more elements than fit into the buffer, simply copy
	 * the tail of the element array into the buffer, otherwise perform
	 * circular write operation.
	 */
	if numElems >= n {
		idx := numElems - n
		b.mutex.Lock()
		copy(values, elems[idx:numElems])
		b.pointer = 0
		b.mutex.Unlock()
	} else {
		b.mutex.Lock()
		ptr := b.pointer
		ptrInc := ptr + numElems

		/*
		 * Check whether the write operation stays within the array bounds.
		 */
		if ptrInc < n {
			copy(values[ptr:ptrInc], elems)
			b.pointer = ptrInc
		} else {
			head := ptrInc - n
			tail := n - ptr
			copy(values[ptr:n], elems[0:tail])
			copy(values[0:head], elems[tail:numElems])
			b.pointer = head
		}

		b.mutex.Unlock()
	}

}

/*
 * Returns the size of b buffer.
 */
func (b *Buffer[T]) Length() int {
	vals := b.values
	n := len(vals)
	return n
}

/*
 * Retrieve all elements from the circular buffer.
 */
func (b *Buffer[T]) Retrieve(buf []T) error {
	values := b.values
	n := len(values)
	m := len(buf)

	/*
	 * Ensure the target buffer is of equal size.
	 */
	if n != m {
		return fmt.Errorf("%s", "Target buffer must be of the same size as source buffer.")
	} else {
		b.mutex.RLock()
		ptr := b.pointer
		tailSize := n - ptr
		copy(buf[0:tailSize], values[ptr:n])
		copy(buf[tailSize:n], values[0:ptr])
		b.mutex.RUnlock()
		return nil
	}

}
func (b *Buffer[T]) At(n int) *T {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	length := b.Length()
	if length == 0 {
		return nil
	}
	if length < n {
		return nil
	}

	index := (b.pointer + n) % length
	return &b.values[index]
}

/*
 * Creates a circular buffer of a certain size.
 */
func CreateBuffer[T any](size int) *Buffer[T] {
	values := make([]T, size)
	m := sync.RWMutex{}

	/*
	 * Create circular buffer.
	 */
	buf := Buffer[T]{
		mutex:   m,
		values:  values,
		pointer: 0,
	}

	return &buf
}
