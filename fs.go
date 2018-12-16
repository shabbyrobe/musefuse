package musefuse

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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

type FS struct {
	root      *dirNode
	entries   []*FileEntry
	failed    []*FileEntry
	handles   *handleMap
	nextInode uint64
}

func NewFS() *FS {
	fs := &FS{
		root:      newDirNode(1, ""),
		nextInode: 2,
		handles:   newHandleMap(),
	}
	return fs
}

func (fs *FS) Root() (fs.Node, error) {
	return fs.root, nil
}

func (fs *FS) inode() uint64 {
	next := fs.nextInode
	fs.nextInode++
	return next
}

func (fs *FS) AddAudio(entry *FileEntry) error {
	fs.entries = append(fs.entries, entry)

	if entry.Err != "" {
		fs.failed = append(fs.failed, entry)
		title := fmt.Sprintf("%s.%d", trimExt(filepath.Base(entry.File.Path), ""), rand.Int63())

		if err := fs.addNode(title, entry, "failed"); err != nil {
			return err
		}

	} else if entry.Metadata != nil {
		added := false

		if entry.Metadata.Artist != "" && entry.Metadata.Title != "" {
			added = true

			if err := fs.addNode(entry.Metadata.Title, entry, "artist", entry.Metadata.Artist); err != nil {
				return err
			}

			if entry.Metadata.Album != "" {
				albumArtist := entry.Metadata.AlbumArtist
				if albumArtist == "" {
					albumArtist = entry.Metadata.Artist
				}

				var title string
				if entry.Metadata.Disc > 0 && (entry.Metadata.Discs > 1 || entry.Metadata.Discs == 0) && entry.Metadata.Track > 0 {
					title = fmt.Sprintf("%02d-%02d %s", entry.Metadata.Disc, entry.Metadata.Track, entry.Metadata.Title)
				} else if entry.Metadata.Track > 0 {
					title = fmt.Sprintf("%02d %s", entry.Metadata.Track, entry.Metadata.Title)
				} else {
					title = entry.Metadata.Title
				}

				if err := fs.addNode(title, entry, "artistalbum", albumArtist, entry.Metadata.Album); err != nil {
					return err
				}
			}

			if entry.Metadata.Year > 0 {
				if err := fs.addNode(
					entry.Metadata.Title,
					entry,
					"year", strconv.FormatInt(int64(entry.Metadata.Year), 10), entry.Metadata.Artist,
				); err != nil {
					return err
				}
			}

			if entry.Metadata.Genre != "" {
				if err := fs.addNode(
					entry.Metadata.Title,
					entry,
					"genre", entry.Metadata.Genre, entry.Metadata.Artist,
				); err != nil {
					return err
				}
			}
		}

		if !added {
			var title = entry.Metadata.Title
			if title == "" {
				title = trimExt(filepath.Base(entry.File.Path), "")
			}

			title = fmt.Sprintf("%s.%d", title, rand.Int63())
			if err := fs.addNode(title, entry, "unsorted"); err != nil {
				return err
			}
		}
	}

	return nil
}

func (fs *FS) lookup(name string) fs.Node {
	dir := fs.root
	name = strings.Trim(name, string(filepath.Separator))
	if name == "" {
		return dir
	}

	parts := strings.Split(name, string(filepath.Separator))
	last := len(parts) - 1
	if last < 0 {
		return nil
	}
	fname := parts[last]

	for _, part := range parts[:len(parts)-1] {
		next, ok := dir.index[part]
		if !ok {
			return nil
		} else if nextDir, ok := next.(*dirNode); ok {
			dir = nextDir
		} else {
			return nil
		}
	}

	node := dir.index[fname]
	if node != nil {
		return node
	}
	return nil
}

func (fs *FS) addNode(name string, entry *FileEntry, path ...string) error {
	ext := filepath.Ext(entry.File.Path)

	dir := fs.root
	for _, part := range path {
		part = sanitisePart.ReplaceAllString(part, "_")

		next, ok := dir.index[part]
		if !ok {
			nextDir := newDirNode(fs.inode(), part)
			dir.addDir(nextDir)
			dir = nextDir

		} else if nextDir, ok := next.(*dirNode); ok {
			dir = nextDir

		} else {
			return fmt.Errorf("musefuse: can't replace dir with file")
		}
	}

	{ // Ensure name is unique:
		baseName := name
		ver := 2
		for {
			if _, ok := dir.index[name]; !ok {
				break
			}
			name = fmt.Sprintf("%s v%d", baseName, ver)
			ver++
		}
	}

	name = sanitisePart.ReplaceAllString(name, "_")
	name += ext
	dir.addFile(newFileNode(fs.handles, fs.inode(), name, entry))

	return nil
}

// Remove ascii control and unsupported filename chars:
// https://superuser.com/questions/358855
var sanitisePart = regexp.MustCompile(`[\x00-\x1F\x7F\\/"?:\*<>\|]`)

func trimExt(file string, trim string) string {
	if trim != "" {
		if trim[0] != '.' {
			return file
		}
		return strings.TrimSuffix(file, trim)
	}
	ext := filepath.Ext(file)
	if ext == "" {
		return file
	}
	return file[0 : len(file)-len(ext)]
}
