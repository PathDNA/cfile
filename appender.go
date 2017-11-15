package cfile

import (
	"io"
	"os"
)

// Appender returns an io.WriteCloser that can be used with any active Readers.
func (f *File) Appender() io.WriteCloser {
	f.mux.RLock()
	f.amux.Lock()
	f.wg.Add(1)
	f.f.Seek(0, io.SeekEnd)
	return &appender{f: f}
}

type appender struct {
	f  *File
	sz int64
}

func (a *appender) Write(b []byte) (n int, err error) {
	if a.f == nil {
		return 0, os.ErrClosed
	}
	n, err = a.f.f.Write(b)
	a.sz += int64(n)
	return
}

func (a *appender) Close() (err error) {
	if a.f == nil {
		return os.ErrClosed
	}

	if a.f.SyncAfterWriterClose {
		err = a.f.f.Sync()
	}

	a.f.sz.Add(a.sz)

	a.f.wg.Done()
	a.f.amux.Unlock()
	a.f.mux.RUnlock()
	a.f = nil

	return
}
