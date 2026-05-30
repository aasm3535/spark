package terminal

// RingBuffer is a fixed-size circular buffer for terminal scrollback.
// Avoids slice reallocation and GC pressure compared to append+trim.
type RingBuffer struct {
	data  [][]Cell
	start int
	size  int
}

// NewRingBuffer creates a ring buffer with given capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		data: make([][]Cell, capacity),
	}
}

// Push adds a line to the buffer, overwriting oldest if full.
func (rb *RingBuffer) Push(line []Cell) {
	end := (rb.start + rb.size) % len(rb.data)
	rb.data[end] = line
	if rb.size < len(rb.data) {
		rb.size++
	} else {
		rb.start = (rb.start + 1) % len(rb.data)
	}
}

// Get retrieves line at logical index (0 = oldest).
func (rb *RingBuffer) Get(index int) []Cell {
	if index < 0 || index >= rb.size {
		return nil
	}
	actual := (rb.start + index) % len(rb.data)
	return rb.data[actual]
}

// Len returns current number of lines stored.
func (rb *RingBuffer) Len() int {
	return rb.size
}

// Cap returns maximum capacity.
func (rb *RingBuffer) Cap() int {
	return len(rb.data)
}

// Clear removes all lines.
func (rb *RingBuffer) Clear() {
	rb.start = 0
	rb.size = 0
}
