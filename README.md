# cfile [![GoDoc](https://godoc.org/github.com/PathDNA/cfile?status.svg)](https://godoc.org/github.com/PathDNA/cfile) [![Build Status](https://travis-ci.org/PathDNA/cfile.svg?branch=master)](https://travis-ci.org/PathDNA/cfile) [![Coverage](https://gocover.io/_badge/github.com/PathDNA/cfile)](https://gocover.io/github.com/PathDNA/cfile)

Concurrent File Reader/Appender/Writer over a single fd.

## Features

* Concurrent multiple readers.

* Append while concurrently reading.

* Only 1 system fd per file.

## Example

```go
f, err := cfile.New("something.file", 0644)
handle(err)
defer f.Close()

for i := 0; i < 10; i++ {
	go func() {
		r := f.Reader()
		defer r.Close()
		// use r
	}()
}

a := f.Appender()
defer a.Close()

a.Write([]byte("hello world"))
```

## License

This project is released under the MIT. See [LICENCE](LICENCE) for more details.
