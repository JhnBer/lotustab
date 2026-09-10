package export

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type chunker struct {
	dir       string
	path      string
	baseName  string
	ext       string
	gzipOn    bool
	chunkSize int

	chunk       int
	rowsInChunk int
	files       []string
	bytes       int64

	file      *os.File
	partPath  string
	finalPath string
	buf       *bufio.Writer
	gz        *gzip.Writer
	out       io.Writer
}

func newChunker(dir, path, baseName, ext string, gzipOn bool, chunkSize int) *chunker {
	if chunkSize < 0 {
		chunkSize = 0
	}
	return &chunker{
		dir:       dir,
		path:      path,
		baseName:  baseName,
		ext:       ext,
		gzipOn:    gzipOn,
		chunkSize: chunkSize,
	}
}

func (c *chunker) Writer() io.Writer {
	return c.out
}

func (c *chunker) Chunk() int { return c.chunk }

func (c *chunker) Bytes() int64 { return c.bytes }

func (c *chunker) Files() []string { return c.files }

func (c *chunker) ShouldRotate() bool {
	return c.chunkSize > 0 && c.rowsInChunk >= c.chunkSize
}

func (c *chunker) AddRows(n int) {
	c.rowsInChunk += n
}

func (c *chunker) Open() error {
	return c.openNext()
}

func (c *chunker) Rotate() error {
	if err := c.closeCurrent(true); err != nil {
		return err
	}
	c.rowsInChunk = 0
	return c.openNext()
}

func (c *chunker) Close() error {
	return c.closeCurrent(true)
}

func (c *chunker) Abort() {
	_ = c.closeCurrent(false)
}

func (c *chunker) DeleteChunks() {
	for _, path := range c.files {
		_ = os.Remove(path)
	}
	c.files = nil
}

func (c *chunker) openNext() error {
	c.chunk++
	finalPath, err := c.nextPath()
	if err != nil {
		return err
	}
	partPath := finalPath + ".part"
	f, err := os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	cw := &countWriter{w: f, n: &c.bytes}
	var out io.Writer = cw
	var gz *gzip.Writer
	if c.gzipOn {
		gz = gzip.NewWriter(cw)
		out = gz
	}
	c.file = f
	c.partPath = partPath
	c.finalPath = finalPath
	c.gz = gz
	c.buf = bufio.NewWriterSize(out, 1<<20)
	c.out = c.buf
	return nil
}

func (c *chunker) nextPath() (string, error) {
	if c.chunkSize <= 0 && strings.TrimSpace(c.path) != "" {
		return c.path, nil
	}
	dir := c.dir
	if dir == "" {
		if c.path != "" {
			dir = filepath.Dir(c.path)
		} else {
			return "", fmt.Errorf("output directory is required")
		}
	}
	name := fmt.Sprintf("%s_%03d%s", c.baseName, c.chunk, c.ext)
	return filepath.Join(dir, name), nil
}

func (c *chunker) closeCurrent(keep bool) error {
	if c.file == nil {
		return nil
	}
	var err error
	if c.buf != nil {
		if e := c.buf.Flush(); e != nil && err == nil {
			err = e
		}
	}
	if c.gz != nil {
		if e := c.gz.Close(); e != nil && err == nil {
			err = e
		}
	}
	if e := c.file.Close(); e != nil && err == nil {
		err = e
	}
	part, final := c.partPath, c.finalPath
	c.file = nil
	c.buf = nil
	c.gz = nil
	c.out = nil
	c.partPath = ""
	c.finalPath = ""
	if !keep {
		_ = os.Remove(part)
		return err
	}
	if err != nil {
		_ = os.Remove(part)
		return err
	}
	if e := os.Rename(part, final); e != nil {
		_ = os.Remove(part)
		return e
	}
	c.files = append(c.files, final)
	return nil
}

type countWriter struct {
	w io.Writer
	n *int64
}

func (c *countWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	*c.n += int64(n)
	return n, err
}
