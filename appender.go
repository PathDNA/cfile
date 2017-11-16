package cfile

import (
	"io"
	"os"
)

// Appender returns an `*Appender` that can be used with any active Readers.
func (f *File) Appender() *Appender {
	f.mux.RLock()
	f.amux.Lock()
	f.wg.Add(1)
	f.f.Seek(0, io.SeekEnd)
	return &Appender{f: f}
}

type Appender struct {
	f  *File
	sz int64
}

func (a *Appender) Write(b []byte) (n int, err error) {
	if a.f == nil {
		return 0, os.ErrClosed
	}
	n, err = a.f.f.Write(b)
	a.sz += int64(n)
	return
}

func (a *Appender) Sync() (err error) {
	if a.sz == 0 {
		return
	}

	if err = a.f.f.Sync(); err != nil {
		return
	}

	a.f.sz.Add(a.sz)
	a.sz = 0
	return
}

func (a *Appender) Close() (err error) {
	if a.f == nil {
		return os.ErrClosed
	}

	err = a.Sync()

	a.f.wg.Done()
	a.f.amux.Unlock()
	a.f.mux.RUnlock()
	a.f = nil

	return
}
