package playlist

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/shabbyrobe/musefuse/playlist/xspf"
)

type Playlist interface {
	Tracks() []Track
	Files() []string
}

type Track interface {
	File() string
}

func LoadPlaylistFile(file string) (Playlist, error) {
	switch strings.ToLower(filepath.Ext(file)) {
	case ".xspf":
		bts, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}

		pls, err := xspf.Unmarshal(bts)
		if err != nil {
			return nil, err
		}

		return &xspfPlaylist{xspf: pls}, nil

	default:
		return nil, fmt.Errorf("playlist: unsupported file %q", file)
	}
}

type xspfPlaylist struct {
	xspf *xspf.Playlist
}

func (x *xspfPlaylist) Files() []string {
	xts := x.xspf.Tracks()
	out := make([]string, 0, len(xts))
	for _, v := range x.xspf.Tracks() {
		f := v.File()
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

func (x *xspfPlaylist) Tracks() []Track {
	xts := x.xspf.Tracks()
	out := make([]Track, len(xts))
	for i, v := range x.xspf.Tracks() {
		out[i] = v
	}
	return out
}
