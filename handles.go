package musefuse

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

type handleMap struct {
	lock    sync.Mutex
	handles map[fuse.HandleID]*handle
	id      fuse.HandleID
}

func newHandleMap() *handleMap {
	return &handleMap{
		handles: map[fuse.HandleID]*handle{},
	}
}

func (hmap *handleMap) add(file *os.File, sz int64) *handle {
	hmap.lock.Lock()
	for {
		if _, ok := hmap.handles[hmap.id]; !ok {
			break
		}
		hmap.id++
	}

	handleID := hmap.id
	h := &handle{id: handleID, handleMap: hmap, file: file, sz: sz}
	hmap.handles[handleID] = h
	hmap.lock.Unlock()
	return h
}

func (hmap *handleMap) destroy(handleID fuse.HandleID) {
	hmap.lock.Lock()
	delete(hmap.handles, handleID)
	hmap.lock.Unlock()
}

func (hmap *handleMap) open(req *fuse.OpenRequest, entry *FileEntry) (*handle, fuse.HandleID, error) {
	f, err := os.Open(filepath.Join(entry.File.Prefix, entry.File.Path))
	if err != nil {
		return nil, 0, err
	}

	st, _ := f.Stat()
	h := hmap.add(f, st.Size())
	return h, h.id, nil
}

type handle struct {
	id        fuse.HandleID
	handleMap *handleMap
	file      *os.File
	sz        int64
}

func (h *handle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	if err := h.file.Close(); err != nil {
		return err
	}
	h.handleMap.destroy(h.id)
	h.file = nil
	return nil
}

func (h *handle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	bufsz := req.Size
	if req.Offset+int64(req.Size) > h.sz {
		bufsz = int(h.sz - req.Offset)
	}
	buf := make([]byte, bufsz)

	n, err := h.file.ReadAt(buf, req.Offset)
	if err != nil {
		return err
	}
	resp.Data = buf[:n]
	return nil
}

var _ fs.Handle = &handle{}
var _ fs.HandleReader = &handle{}
var _ fs.HandleReleaser = &handle{}
