package cfile

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"

	. "github.com/PathDNA/testutils"
)

// verify that we match the io.* interfaces.
var (
	_ io.Seeker     = (*Reader)(nil)
	_ io.ReaderAt   = (*Reader)(nil)
	_ io.ReadCloser = (*Reader)(nil)

	_ io.Seeker      = (*Writer)(nil)
	_ io.WriterAt    = (*Writer)(nil)
	_ io.WriteCloser = (*Writer)(nil)

	_ io.Reader     = (*File)(nil)
	_ io.ReaderFrom = (*File)(nil)
	_ io.ReaderAt   = (*File)(nil)
	_ io.Writer     = (*File)(nil)
	_ io.WriterTo   = (*File)(nil)
	_ io.WriterAt   = (*File)(nil)
	_ io.Closer     = (*File)(nil)
)

var dummyData = bytes.Repeat([]byte("0123456789"), 2)

func Test(t *testing.T) {
	f, err := ioutil.TempFile("", "file-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	if n, err := f.Write(dummyData); FailIf(t, err) || n != len(dummyData) {
		return
	}

	mf, err := FromFile(f)
	if FailIf(t, err) {
		return
	}

	t.Run("Reader Limit", func(t *testing.T) {
		r := mf.Reader()
		a := mf.Appender()
		a.Write(dummyData)
		a.Close()
		b, err := ioutil.ReadAll(r)
		r.Close()
		if err != nil || !bytes.Equal(b, dummyData) {
			t.Errorf("unexpected read (%s | %s): %v", b, dummyData, err)
		}

		r = mf.Reader()
		b, err = ioutil.ReadAll(r)
		r.Close()
		if err != nil || len(b) != len(dummyData)*2 {
			t.Errorf("invalid file state :( (%s | %s): %v", b, dummyData, err)
		}
	})

	t.Run("Concurrent Read Append", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(2)
			i := i
			go func() {
				defer wg.Done()
				wc := mf.Appender()
				defer wc.Close()
				buf := bytes.Repeat([]byte{byte(i) + '0'}, 10)
				if n, err := wc.Write(buf); err != nil || n != len(buf) {
					t.Errorf("%d (%d/%d): %v", i, n, len(buf), err)
					return
				}
			}()

			go func() {
				defer wg.Done()
				buf := make([]byte, len(dummyData)-10)
				r := mf.ReaderAt(10)
				defer r.Close()
				n, err := r.Read(buf)
				if err != nil || n != len(buf) {
					t.Errorf("%d (%d/%d): %v", i, n, len(buf), err)
					return
				}
				if !bytes.Equal(buf[:n], dummyData[10:]) {
					t.Errorf("%s mismatch", buf)
				}
			}()
		}

		wg.Wait()
	})

	var buf bytes.Buffer
	mf.WriteTo(&buf)
	if buf.Len() != (len(dummyData)*2)+(10*10) {
		t.Errorf("unexpected data: %s", buf.Bytes())
	}

	mf.Close() // make sure there are no deadlocks

	if _, err = mf.Read([]byte{0}); err == nil || !strings.Contains(err.Error(), "already closed") {
		t.Errorf("unexpected error: %#+v", err)
	}

	if _, err = mf.Write([]byte{0}); err == nil || !strings.Contains(err.Error(), "already closed") {
		t.Errorf("unexpected error: %#+v", err)
	}
}
