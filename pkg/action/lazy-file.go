package action

import (
	"io"
	"os"
)

type lazyFile struct {
	out    io.WriteCloser
	name   string
	prefix string
}

func (lw *lazyFile) Write(p []byte) (int, error) {
	if lw.out == nil {
		fileName, out, err := makeLogWriter(lw.prefix)
		if err != nil {
			return 0, err
		}

		lw.out = out
		lw.name = fileName
	}

	n, err := lw.out.Write(p)
	if err != nil {
		return n, err
	}

	if file, ok := lw.out.(*os.File); ok {
		if err = file.Sync(); err != nil {
			return n, err
		}
	}

	return n, nil
}

func (lw *lazyFile) Close() {
	if lw.out != nil {
		lw.out.Close()
	}
}
