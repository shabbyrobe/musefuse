package musefuse

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

type FileEntry struct {
	File     FileInfo
	Err      string
	Metadata *Metadata
}

type fileNode struct {
	handleMap *handleMap
	inode     uint64
	name      string
	parent    *dirNode
	entry     *FileEntry
}

func newFileNode(hmap *handleMap, inode uint64, name string, entry *FileEntry) *fileNode {
	return &fileNode{
		handleMap: hmap,
		inode:     inode,
		name:      name,
		entry:     entry,
	}
}

func (file *fileNode) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = file.inode
	a.Mode = 0600
	a.Size = uint64(file.entry.File.Size)
	a.Mtime = file.entry.File.ModTime
	return nil
}

func (file *fileNode) ReadAll(ctx context.Context) ([]byte, error) {
	path := filepath.Join(file.entry.File.Prefix, file.entry.File.Path)
	return ioutil.ReadFile(path)
}

func (file *fileNode) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	if !req.Flags.IsReadOnly() {
		return nil, fuse.Errno(syscall.EACCES)
	}
	resp.Flags |= fuse.OpenKeepCache
	handle, id, err := file.handleMap.open(req, file.entry)
	if err != nil {
		return nil, err
	}
	resp.Handle = id
	return handle, nil
}

type dirNode struct {
	inode   uint64
	name    string
	parent  *dirNode
	files   []*fileNode
	dirs    []*dirNode
	entries []fuse.Dirent
	index   map[string]fs.Node
}

func newDirNode(inode uint64, name string) *dirNode {
	return &dirNode{
		inode: inode,
		name:  name,
		index: map[string]fs.Node{},
	}
}

func (dir *dirNode) addDir(add *dirNode) {
	dir.entries = append(dir.entries, fuse.Dirent{
		Inode: add.inode,
		Name:  add.name,
		Type:  fuse.DT_Dir,
	})
	dir.index[add.name] = add
	dir.dirs = append(dir.dirs, add)
	add.parent = dir
}

func (dir *dirNode) addFile(file *fileNode) {
	dir.entries = append(dir.entries, fuse.Dirent{
		Inode: file.inode,
		Name:  file.name,
		Type:  fuse.DT_File,
	})
	dir.index[file.name] = file
	dir.files = append(dir.files, file)
	file.parent = dir
}

func (dir *dirNode) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = dir.inode
	a.Mode = os.ModeDir | 0700
	return nil
}

func (dir *dirNode) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if node, ok := dir.index[name]; ok {
		return node, nil
	}
	return nil, fuse.ENOENT
}

func (dir *dirNode) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return dir.entries, nil
}
