package db

// NopReadWriter provides a noop reader and writer.
type NopReadWriter struct {
	NopWriter
	NopReader
}

// NopWriter provides a noop writer.
type NopWriter struct{}

// Write implements the io.Writer interface and does nothing.
func (NopWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

// NopReader provides a noop reader.
type NopReader struct{}

// Read implements the io.Reader interface and does nothing.
func (NopReader) Read(b []byte) (int, error) {
	return len(b), nil
}
