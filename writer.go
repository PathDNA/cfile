package cfile

import (
	"io"
	"os"
)

// Writer returns f.WriteAt(-1).
func (f *File) Writer() *Writer {
	return f.WriterAt(-1)
}

// WriterAt acquires a write-lock, seeks to the given offset and returns a writer.
// if off is < 0, it seeks to the end of the file, otherwise it seeks to the off value.
func (f *File) WriterAt(off int64) *Writer {
	f.mux.Lock()
	f.wg.Add(1)
	if off < 0 {
		off, _ = f.f.Seek(0, io.SeekEnd)
	} else {
		f.f.Seek(off, io.SeekStart)
	}

	return &Writer{
		f:   f,
		off: off,
	}
}

// Writer implements `io.Writer`, `io.WriterAt`, `io.Seeker`` and `io.Closer`.
type Writer struct {
	f   *File
	off int64
}

// Write implements `io.Writer`.
func (w *Writer) Write(b []byte) (n int, err error) {
	if w.f == nil {
		return 0, os.ErrClosed
	}
	n, err = w.f.f.Write(b)
	w.off += int64(n)
	return
}

// WriteAt implements `io.WriterAt`.
func (w *Writer) WriteAt(b []byte, off int64) (n int, err error) {
	if w.f == nil {
		return 0, os.ErrClosed
	}
	return w.f.WriteAt(b, off)
}

// Seek implements `io.Seeker`.
func (w *Writer) Seek(offset int64, whence int) (n int64, err error) {
	if w.f == nil {
		return 0, os.ErrClosed
	}
	n, err = w.f.f.Seek(offset, whence)
	w.off = n
	return
}

// Close releases the parent's write-lock.
func (w *Writer) Close() (err error) {
	if w.f == nil {
		return os.ErrClosed
	}

	defer func() {
		w.f.wg.Done()
		w.f.mux.Unlock()
		w.f = nil
	}()

	if w.f.SyncAfterWriterClose {
		if err = w.f.f.Sync(); err != nil {
			return
		}
	}

	var sz int64
	if sz, err = getSize(w.f.f); err != nil {
		return
	}

	w.f.sz.Store(sz)

	return
}
