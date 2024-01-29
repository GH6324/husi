package libcore

import (
	"io"
	"log"
	"os"
	"syscall"

	E "github.com/sagernet/sing/common/exceptions"
)

var husiLogWriter *logWriter
var logWriterDisable = false
var truncateOnStart = true
var guiLogWriter io.Writer

func setupLog(maxSize int64, path string) (err error) {
	if husiLogWriter != nil {
		return
	}

	var f *os.File
	f, err = os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err == nil {
		fd := int(f.Fd())
		if truncateOnStart {
			syscall.Flock(fd, syscall.LOCK_EX)
			// Check if need truncate
			if size, _ := f.Seek(0, io.SeekEnd); size > maxSize {
				// read oldBytes for maxSize
				f.Seek(-maxSize, io.SeekCurrent)
				oldBytes, err := io.ReadAll(f)
				if err == nil {
					// truncate file
					err = f.Truncate(0)
					// write oldBytes
					if err == nil {
						f.Write(oldBytes)
					}
				}
			}
			syscall.Flock(fd, syscall.LOCK_UN)
		}
		// redirect stderr
		syscall.Dup3(fd, int(os.Stderr.Fd()), 0)
	}

	if err != nil {
		log.Println(E.Cause(err, "open log"))
	}

	//
	husiLogWriter = &logWriter{}
	husiLogWriter.writers = []io.Writer{guiLogWriter, f}
	// setup std log
	log.SetFlags(log.LstdFlags | log.LUTC)
	log.SetOutput(husiLogWriter)

	return
}

type logWriter struct {
	writers []io.Writer
}

func (w *logWriter) Write(p []byte) (int, error) {
	if logWriterDisable {
		return len(p), nil
	}

	for _, w := range w.writers {
		if w == nil {
			continue
		}
		if f, ok := w.(*os.File); ok {
			fd := int(f.Fd())
			syscall.Flock(fd, syscall.LOCK_EX)
			f.Write(p)
			syscall.Flock(fd, syscall.LOCK_UN)
		} else {
			w.Write(p)
		}
	}

	return len(p), nil
}

func (w *logWriter) truncate() {
	for _, w := range w.writers {
		if w == nil {
			continue
		}
		if f, ok := w.(*os.File); ok {
			_ = f.Truncate(0)
		}
	}
}

func (w *logWriter) Close() error {
	for _, w := range w.writers {
		if w == nil {
			continue
		}
		if f, ok := w.(*os.File); ok {
			_ = f.Close()
		}
	}
	return nil
}
