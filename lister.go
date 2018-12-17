package musefuse

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/shabbyrobe/golib/pathtools"
	"github.com/shabbyrobe/musefuse/fastwalk"
)

type FileInfo struct {
	Prefix  string
	Path    string
	Size    int64
	ModTime time.Time
	Kind    FileKind
}

type FileKind string

const (
	FileAudio    FileKind = "audio"
	FilePlaylist FileKind = "playlist"
)

// Lister, if you touch that guitar, I'll remove the E-string and garotte you
// with it.
type Lister struct {
	paths            []string
	audioExtIndex    map[string]bool
	playlistExtIndex map[string]bool
}

func NewLister(paths []string, audioExts, playlistExts []string) *Lister {
	audioExtIndex := make(map[string]bool, len(audioExts))
	for _, ext := range audioExts {
		audioExtIndex[strings.ToLower(ext)] = true
	}
	playlistExtIndex := make(map[string]bool, len(playlistExts))
	for _, ext := range playlistExts {
		playlistExtIndex[strings.ToLower(ext)] = true
	}

	return &Lister{
		audioExtIndex:    audioExtIndex,
		playlistExtIndex: playlistExtIndex,
		paths:            paths,
	}
}

// Lister, don't be a gimboid!
var fileGarbagePattern = regexp.MustCompile(`(?i)[/\\].AppleDouble[/\\]`)

func (lister *Lister) List(into []FileInfo) (files []FileInfo, err error) {
	files = into[:0]

	for _, database := range lister.paths {
		database, err = filepath.Abs(database)
		if err != nil {
			return nil, err
		}

		if err := fastwalk.Walk(database, func(path string, typ os.FileMode) error {
			if typ.IsDir() {
				return nil
			}

			var kind FileKind
			if lister.audioExtIndex[strings.ToLower(filepath.Ext(path))] {
				kind = FileAudio
			} else if lister.playlistExtIndex[strings.ToLower(filepath.Ext(path))] {
				kind = FilePlaylist
			} else {
				return nil
			}

			if filepath.Base(path)[0] == '.' {
				return nil
			}
			if fileGarbagePattern.MatchString(path) {
				return nil
			}

			fullPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}

			ok, filePrefix, filePath, err := pathtools.FilepathPrefix(fullPath, database)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("musefuse: could not separate path %q from prefix %q", fullPath, database)
			}

			st, err := os.Stat(fullPath)
			if err != nil {
				return err
			}

			info := FileInfo{
				Kind:    kind,
				Prefix:  filePrefix,
				Path:    filePath,
				ModTime: st.ModTime(),
				Size:    st.Size(),
			}
			files = append(files, info)

			return nil

		}); err != nil {
			return nil, err
		}
	}

	return files, err
}
