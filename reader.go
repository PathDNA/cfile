package cfile

import (
	"io"
	"os"
)

// Reader returns f.SectionReader(0, -1).
func (f *File) Reader() *Reader { return f.SectionReader(0, -1) }

// ReaderAt returns f.SectionReader(off, -1).
func (f *File) ReaderAt(off int64) *Reader { return f.SectionReader(off, -1) }

// SectionReader returns a SectionReader that reads from f
// starting at offset off and stops with EOF after n bytes.
// if n - 1, it will read the entire file.
func (f *File) SectionReader(off, n int64) *Reader {
	sz := f.Size()
	if n == -1 || off+n > sz {
		n = sz - off
	}

	f.mux.RLock()
	f.wg.Add(1)
	return &Reader{
		f: f,
		r: io.NewSectionReader(f.f, off, n),
	}
}

// Reader implements `io.Reader`, `io.ReaderAt`, `io.Seeker` and `io.Closer`.
type Reader struct {
	f *File
	r *io.SectionReader
}

// Read implements `io.Read`.
func (r *Reader) Read(b []byte) (int, error) {
	if r.f == nil {
		return 0, os.ErrClosed
	}

	return r.r.Read(b)
}

// ReadAt implements `io.ReaderAt`.
func (r *Reader) ReadAt(b []byte, off int64) (int, error) {
	if r.f == nil {
		return 0, os.ErrClosed
	}

	return r.r.ReadAt(b, off)
}

// Seek implements `io.Seeker`.
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	if r.f == nil {
		return 0, os.ErrClosed
	}

	return r.r.Seek(offset, whence)
}

func (r *Reader) Size() int64 {
	return r.r.Size()
}

// Close releases the parent's read-lock.
func (r *Reader) Close() error {
	if r.f == nil {
		return os.ErrClosed
	}
	r.f.wg.Done()
	r.f.mux.RUnlock()
	r.f = nil
	return nil
}
