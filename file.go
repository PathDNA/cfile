package cfile

// TODO:
// * explore file locking
// * use mmap, but that would make appending/trunc a lot harder.

import (
	"io"
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"
)

var (
	_ io.Reader     = (*File)(nil)
	_ io.ReaderFrom = (*File)(nil)
	_ io.ReaderAt   = (*File)(nil)
	_ io.Writer     = (*File)(nil)
	_ io.WriterTo   = (*File)(nil)
	_ io.WriterAt   = (*File)(nil)
	_ io.Closer     = (*File)(nil)
)

// New opens `fp`` with `os.O_CREATE|os.O_RDWR`` and the given permissions.
func New(fp string, perm os.FileMode) (*File, error) {
	f, err := os.OpenFile(fp, os.O_CREATE|os.O_RDWR, perm)

	if err != nil {
		return nil, err
	}

	mf, err := FromFile(f)
	if err != nil {
		f.Close()
	}

	return mf, err
}

// TempFile is a convenience wrapper for `ioutil.TempFile`.
// Remember to clean up by calling `os.Remove(f.Name())` when you're done.
func TempFile(dir, prefix string) (*File, error) {
	f, err := ioutil.TempFile(dir, prefix)
	if err != nil {
		return nil, err
	}

	mf, err := FromFile(f)
	if err != nil {
		f.Close()
		os.Remove(f.Name())
	}

	return mf, err
}

// FromFile returns a `*File` from an `*os.File`.
// Using `f.Writer` requires that the file to *not* be opened with os.O_APPEND.
// On success, the returned `*File` will handle closing the `*os.File`.
func FromFile(f *os.File) (*File, error) {
	sz, err := getSize(f)

	if err != nil {
		return nil, err
	}

	return &File{
		f:    f,
		size: sz,
	}, nil
}

// File is an `*os.File` wrapper that allows multiple readers or one writer or appender on a single file descriptor.
type File struct {
	f *os.File

	mux  sync.RWMutex
	amux sync.Mutex

	wg sync.WaitGroup

	size int64

	SyncAfterWriterClose bool // if set to true, calling `Writer.Close()`, will call `*os.File.Sync()`.
}

// ReadAt implements `io.Read`.
// Note that it will always start at offset 0.
func (f *File) Read(b []byte) (n int, err error) {
	r := f.ReaderAt(0)
	n, err = r.Read(b)
	r.Close()
	return
}

// ReadAt implements `io.ReaderAt`.
func (f *File) ReadAt(b []byte, off int64) (n int, err error) {
	r := f.ReaderAt(off)
	n, err = r.Read(b)
	r.Close()
	return
}

// WriteTo implements `io.WriterTo`.
func (f *File) WriteTo(w io.Writer) (n int64, err error) {
	r := f.ReaderAt(0)
	n, err = io.Copy(w, r)
	r.Close()
	return
}

// ReadFrom implements `io.ReaderFrom`.
// Appends to the end of the file.
func (f *File) ReadFrom(rd io.Reader) (n int64, err error) {
	wr := f.Appender()
	n, err = io.Copy(wr, rd)
	wr.Close()
	return
}

// WriteAt implements `io.WriterAt`.
func (f *File) WriteAt(b []byte, off int64) (n int, err error) {
	wr := f.WriterAt(off)
	n, err = wr.Write(b)
	wr.Close()
	return
}

// Write wraps `f.Appender().Write` for convenience.
// If expecting multiple `Write` calls, use `f.Appender()`` directly.
func (f *File) Write(b []byte) (n int, err error) {
	a := f.Appender()
	n, err = a.Write(b)
	a.Close()
	return
}

// Truncate truncates the underlying `*os.File` to the specific size.
func (f *File) Truncate(sz int64) (err error) {
	f.mux.Lock()
	if err = f.f.Truncate(sz); err == nil {
		f.setSize(sz)
	}
	f.mux.Unlock()
	return
}

// Size returns the current file size.
// the size is cached after each writer is closed, so it doesn't call Stat().
func (f *File) Size() int64 { return atomic.LoadInt64(&f.size) }

// Stat calls the underlying `*os.File.Stat()`.
// Will block if there are any active appenders or writers.
func (f *File) Stat() (fi os.FileInfo, err error) {
	f.mux.Lock()
	fi, err = f.f.Stat()
	f.mux.Unlock()
	return
}

// Name returns the name of the file.
func (f *File) Name() string {
	return f.f.Name()
}

// With acquires a write lock and calls fn with the underlying `*os.File` and returns any errors it returns.
func (f *File) With(fn func(*os.File) error) error {
	f.mux.Lock()
	defer f.mux.Unlock()
	return fn(f.f)
}

// ForceClose will close the underlying `*os.File` without waiting for any active readers/writer.
func (f *File) ForceClose() error {
	return f.f.Close()
}

// Close waits for all the active readers/writer to finish before closing the underlying `*os.File`.
func (f *File) Close() error {
	f.wg.Wait()

	f.mux.Lock()
	err := f.f.Close()
	f.mux.Unlock()

	return err
}

func (f *File) setSize(sz int64)          { atomic.StoreInt64(&f.size, sz) }
func (f *File) addSize(delta int64) int64 { return atomic.AddInt64(&f.size, delta) }

func getSize(f *os.File) (int64, error) {
	st, err := f.Stat()

	if err != nil {
		return 0, err
	}

	if !st.Mode().IsRegular() {
		return 0, os.ErrInvalid
	}

	return st.Size(), nil
}
