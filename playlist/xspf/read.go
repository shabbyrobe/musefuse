package xspf

import (
	"encoding/xml"
)

func Unmarshal(bts []byte) (*Playlist, error) {
	var pls Playlist

	if err := xml.Unmarshal(bts, &pls); err != nil {
		return nil, err
	}

	return &pls, nil
}
