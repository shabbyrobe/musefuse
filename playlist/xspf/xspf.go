package xspf

import (
	"encoding/xml"
	"net/url"
	"strconv"
	"time"
)

type Playlist struct {
	Version int `xml:"version,attr,omitempty"`

	// A human-readable title for the playlist. xspf:playlist elements MAY contain exactly one.
	Title string `xml:"title,omitempty"`

	// Human-readable name of the entity (author, authors, group, company, etc)
	// that authored the playlist. xspf:playlist elements MAY contain exactly
	// one.
	Creator string `xml:"creator,omitempty"`

	// A human-readable comment on the playlist. This is character data, not
	// HTML, and it may not contain markup. xspf:playlist elements MAY contain
	// exactly one.
	Annotation string `xml:"annotation,omitempty"`

	// URI of a web page to find out more about this playlist. Likely to be
	// homepage of the author, and would be used to find out more about the
	// author and to find more playlists by the author. xspf:playlist elements
	// MAY contain exactly one.
	Info string `xml:"info,omitempty"`

	// Source URI for this playlist. xspf:playlist elements MAY contain exactly one.
	Location string `xml:"location,omitempty"`

	// Canonical ID for this playlist. Likely to be a hash or other
	// location-independent name. MUST be a legal URI. xspf:playlist elements MAY
	// contain exactly one.
	Identifier string `xml:"identifier,omitempty"`

	// URI of an image to display in the absence of a //playlist/trackList/image
	// element. xspf:playlist elements MAY contain exactly one.
	Image string `xml:"image,omitempty"`

	// Creation date (not last-modified date) of the playlist, formatted as a XML
	// schema dateTime. xspf:playlist elements MAY contain exactly one.
	//
	// A sample date is "2005-01-08T17:10:47-05:00"
	//
	// In the absence of a timezone, the element MAY be assumed to use Coordinated
	// Universal Time (UTC, sometimes called "Greenwich Mean Time").
	Date string `xml:"date,omitempty"`

	// URI of a resource that describes the license under which this playlist was
	// released. xspf:playlist elements may contain zero or one license element.
	License string `xml:"license,omitempty"`

	// An ordered list of URIs. The purpose is to satisfy licenses allowing
	// modification but requiring attribution. If you modify such a playlist,
	// move its //playlist/location or //playlist/identifier element to the top
	// of the items in the //playlist/attribution element. xspf:playlist
	// elements MAY contain exactly one xspf:attribution element.
	//
	// Such a list can grow without limit, so as a practical matter we suggest
	// deleting ancestors more than ten generations back.
	Attribution []string `xml:"attribution,omitempty"`

	// The link element allows XSPF to be extended without the use of XML
	// namespaces. xspf:playlist elements MAY contain zero or more link
	// elements.
	Link []Link `xml:">link"`

	// The meta element allows metadata fields to be added to XSPF.
	// xspf:playlist elements MAY contain zero or more meta elements.
	Meta []Meta `xml:">meta"`

	// The extension element allows non-XSPF XML to be included in XSPF
	// documents. The purpose is to allow nested XML, which the meta and link
	// elements do not. xspf:playlist elements MAY contain zero or more
	// extension elements.
	Extension []Extension `xml:">extension"`

	// Ordered list of xspf:track elements to be rendered. The sequence is a hint,
	// not a requirement; renderers are advised to play tracks from top to bottom
	// unless there is an indication otherwise.
	//
	// If an xspf:track element cannot be rendered, a user-agent MUST skip to the
	// next xspf:track element and MUST NOT interrupt the sequence.
	//
	// xspf:playlist elements MUST contain one and only one trackList element. The
	// trackList element my be empty.
	TrackList TrackList `xml:"trackList,omitempty"`
}

func (p Playlist) Tracks() []Track {
	return p.TrackList.Tracks
}

type TrackList struct {
	Tracks []Track `xml:"track"`
}

type Track struct {
	// URI of resource to be rendered. Probably an audio resource, but MAY be
	// any type of resource with a well-known duration, such as video, a SMIL
	// document, or an XSPF document. The duration of the resource defined in
	// this element defines the duration of rendering. xspf:track elements MAY
	// contain zero or more location elements, but a user-agent MUST NOT render
	// more than one of the named resources.
	Locations []string `xml:"location,omitempty"`

	// Canonical ID for this resource. Likely to be a hash or other
	// location-independent name, such as a MusicBrainz identifier. MUST be a legal
	// URI. xspf:track elements MAY contain zero or more identifier elements.
	Identifier string `xml:"identifier,omitempty"`

	// Human-readable name of the track that authored the resource which defines
	// the duration of track rendering. This value is primarily for fuzzy lookups,
	// though a user-agent may display it. xspf:track elements MAY contain exactly
	// one.
	Title string `xml:"title,omitempty"`

	// Human-readable name of the entity (author, authors, group, company, etc)
	// that authored the resource which defines the duration of track rendering.
	// This value is primarily for fuzzy lookups, though a user-agent may display
	// it. xspf:track elements MAY contain exactly one.
	Creator string `xml:"creator,omitempty"`

	// A human-readable comment on the track. This is character data, not HTML, and
	// it may not contain markup. xspf:track elements MAY contain exactly one.
	Annotation string `xml:"annotation,omitempty"`

	// URI of a place where this resource can be bought or more info can be found.
	// xspf:track elements MAY contain exactly one.
	Info string `xml:"info,omitempty"`

	// URI of an image to display for the duration of the track. xspf:track
	// elements MAY contain exactly one.
	Image string `xml:"image,omitempty"`

	// Human-readable name of the collection from which the resource which defines the
	// duration of track rendering comes. For a song originally published as a part of
	// a CD or LP, this would be the title of the original release. This value is
	// primarily for fuzzy lookups, though a user-agent may display it. xspf:track
	// elements MAY contain exactly one.
	Album string `xml:"album,omitempty"`

	// Integer with value greater than zero giving the ordinal position of the
	// media on the xspf:album. This value is primarily for fuzzy lookups, though a
	// user-agent may display it. xspf:track elements MAY contain exactly one. It
	// MUST be a valid XML Schema nonNegativeInteger.
	TrackNum int `xml:"trackNum,omitEmpty"`

	// The time to render a resource, in milliseconds. It MUST be a valid XML Schema
	// nonNegativeInteger. This value is only a hint — different XSPF generators will
	// generate slightly different values. A user-agent MUST NOT use this value to
	// determine the rendering duration, since the data will likely be low quality.
	// xspf:track elements MAY contain exactly one duration element.
	Duration MillisecondDuration `xml:"duration,omitEmpty"`
}

func (t Track) MainLocation() string {
	if len(t.Locations) > 0 {
		return t.Locations[0]
	}
	return ""
}

// File returns the first Location if it is a 'file://' location.
func (t Track) File() (name string) {
	if len(t.Locations) == 0 {
		return ""
	}

	u, err := url.Parse(t.Locations[0])
	if err != nil {
		return ""
	}

	if u.Scheme != "file" {
		return ""
	}
	if u.Host != "" {
		return ""
	}

	return u.Path
}

type Link struct {
	Rel     string `xml:"rel,attr,omitempty"`
	Content string `xml:",chardata"`
}

type Meta struct {
	Rel     string `xml:"rel,attr,omitempty"`
	Content string `xml:",chardata"`
}

type Extension struct {
	Application string `xml:"application,omitempty"`
	Any
}

type Any struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Elems   []Any      `xml:",any"`
}

type MillisecondDuration time.Duration

func (ms MillisecondDuration) MarshalXML(enc *xml.Encoder, start xml.StartElement) error {
	return enc.Encode(strconv.FormatInt(int64(ms)/int64(time.Millisecond), 10))
}

func (ms *MillisecondDuration) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var f int64
	if err := d.DecodeElement(&f, &start); err != nil {
		return err
	}
	*ms = MillisecondDuration(time.Duration(f) * time.Millisecond)
	return nil
}
