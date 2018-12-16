package musefuse

import (
	"strings"

	"github.com/dhowden/tag"
)

var AudioExtensions = []string{".oga", ".mp3", ".ogg", ".opus", ".flac", ".mp4", ".m4a", ".alac", ".aac"}
var PlaylistExtensions = []string{".xspf"}

// Metadata is a structured representation of the tag.Metadata interface for
// serialisation.
type Metadata struct {
	Format      tag.Format
	FileType    tag.FileType
	Title       string
	Album       string
	Artist      string
	AlbumArtist string
	Composer    string
	Year        int
	Genre       string
	Lyrics      string
	Comment     string

	Track  int
	Tracks int

	Disc  int
	Discs int

	// Picture returns a picture, or nil if not available.
	Picture *tag.Picture

	// Raw returns the raw mapping of retrieved tag names and associated values.
	// NB: tag/atom names are not standardised between formats.
	Raw map[string]interface{}
}

func (meta *Metadata) ToTag() tag.Metadata {
	return &metadataAdapter{meta}
}

func MetadataFromTag(tagData tag.Metadata) *Metadata {
	if tagData == nil {
		return nil
	}

	m := &Metadata{
		Format:      tagData.Format(),
		FileType:    tagData.FileType(),
		Title:       strings.TrimSpace(tagData.Title()),
		Album:       strings.TrimSpace(tagData.Album()),
		Artist:      strings.TrimSpace(tagData.Artist()),
		AlbumArtist: strings.TrimSpace(tagData.AlbumArtist()),
		Composer:    strings.TrimSpace(tagData.Composer()),
		Genre:       strings.TrimSpace(tagData.Genre()),
		Year:        tagData.Year(),
		Picture:     tagData.Picture(),
		Lyrics:      tagData.Lyrics(),
		Comment:     strings.TrimSpace(tagData.Comment()),
	}
	m.Track, m.Tracks = tagData.Track()
	m.Disc, m.Discs = tagData.Disc()

	// FIXME: deep copy:
	srcRaw := tagData.Raw()
	m.Raw = make(map[string]interface{}, len(srcRaw))
	for k, v := range srcRaw {
		m.Raw[k] = v
	}

	return m
}

type metadataAdapter struct {
	inner *Metadata
}

func (meta *metadataAdapter) Format() tag.Format          { return meta.inner.Format }
func (meta *metadataAdapter) FileType() tag.FileType      { return meta.inner.FileType }
func (meta *metadataAdapter) Title() string               { return meta.inner.Title }
func (meta *metadataAdapter) Album() string               { return meta.inner.Album }
func (meta *metadataAdapter) Artist() string              { return meta.inner.Artist }
func (meta *metadataAdapter) AlbumArtist() string         { return meta.inner.AlbumArtist }
func (meta *metadataAdapter) Composer() string            { return meta.inner.Composer }
func (meta *metadataAdapter) Year() int                   { return meta.inner.Year }
func (meta *metadataAdapter) Genre() string               { return meta.inner.Genre }
func (meta *metadataAdapter) Track() (int, int)           { return meta.inner.Track, meta.inner.Tracks }
func (meta *metadataAdapter) Disc() (int, int)            { return meta.inner.Disc, meta.inner.Discs }
func (meta *metadataAdapter) Picture() *tag.Picture       { return meta.inner.Picture }
func (meta *metadataAdapter) Lyrics() string              { return meta.inner.Lyrics }
func (meta *metadataAdapter) Comment() string             { return meta.inner.Comment }
func (meta *metadataAdapter) Raw() map[string]interface{} { return meta.inner.Raw }
