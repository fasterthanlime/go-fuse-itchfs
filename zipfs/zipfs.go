// Copyright 2016 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zipfs

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/itchio/wharf/eos"
)

type ZipFile struct {
	*zip.File
}

func (f *ZipFile) Stat(out *fuse.Attr) {
	// TODO - do something intelligent with timestamps.
	// out.Mode = fuse.S_IFREG | 0444
	out.Mode = fuse.S_IFREG | uint32(f.Mode()&0777) | 0666
	out.Size = uint64(f.File.UncompressedSize)
}

func (f *ZipFile) Data() []byte {
	zf := (*f)
	rc, err := zf.Open()
	if err != nil {
		panic(err)
	}
	dest := bytes.NewBuffer(make([]byte, 0, f.UncompressedSize))

	_, err = io.CopyN(dest, rc, int64(f.UncompressedSize))
	if err != nil {
		panic(err)
	}
	return dest.Bytes()
}

// NewZipTree creates a new file-system for the zip file named name.
func NewZipTree(name string) (map[string]MemFile, error) {
	fr, err := eos.Open(name)
	if err != nil {
		return nil, err
	}

	stats, _ := fr.Stat()

	r, err := zip.NewReader(fr, stats.Size())
	if err != nil {
		return nil, err
	}

	out := map[string]MemFile{}
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		n := filepath.Clean(f.Name)

		zf := &ZipFile{f}
		out[n] = zf
	}
	return out, nil
}

func NewArchiveFileSystem(name string) (root nodefs.Node, err error) {
	var files map[string]MemFile

	files, err = NewZipTree(name)

	if err != nil {
		return nil, err
	}

	mfs := NewMemTreeFs(files)
	mfs.Name = fmt.Sprintf("fs(%s)", name)
	return mfs.Root(), nil
}
