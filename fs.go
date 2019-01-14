package musefuse

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"bazil.org/fuse/fs"
)

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

	name = sanitisePart.ReplaceAllString(name, "_")
	baseName := name
	name += ext

	{ // Ensure name is unique:
		ver := 2
		for {
			if _, ok := dir.index[name]; !ok {
				break
			}
			name = fmt.Sprintf("%s v%d%s", baseName, ver, ext)
			ver++
		}
	}

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
