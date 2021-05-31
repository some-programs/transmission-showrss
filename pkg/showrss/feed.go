package showrss

import (
	"encoding/xml"
	"fmt"
	"log"
	"strings"
	"time"
)

// RSS .
type rss struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel
}

type Channel struct {
	XMLName  xml.Name  `xml:"channel"`
	Title    string    `xml:"title"`
	TTL      int       `xml:"ttl"`
	Episodes []Episode `xml:"item"`

	URL string // request url

}

func (c Channel) TTLDuration() time.Duration {
	return time.Duration(c.TTL) * time.Minute
}

// Enclosure .
type Enclosure struct {
	MimeType string `xml:"type,attr" json:"type"`
	URL      string `xml:"url,attr" json:"attr"`
}

// Episode .
type Episode struct {
	Title      string      `xml:"title" json:"title"`
	InfoHash   string      `xml:"info_hash" json:"info_hash"`
	Enclosures []Enclosure `xml:"enclosure" json:"enclosures"`
	ShowID     int         `xml:"show_id" json:"show_id"`
	ExternalID int         `xml:"external_id" json:"external_id"`
	ShowName   string      `xml:"show_name" json:"show_name"`
	EpisodeID  string      `xml:"episode_id" json:"episode_id"`
	RawTitle   string      `xml:"raw_title" json:"raw_title"`
}

func (i Episode) String() string {
	return fmt.Sprintf("%s (%s)", i.InfoHash, i.Title)
}

func (i Episode) Key() []byte {
	if i.InfoHash == "" {
		log.Fatal("InfoHash is empty, no key possible")
	}
	return []byte(i.InfoHash)
}

func (i Episode) URL() string {
	for _, e := range i.Enclosures {
		if e.MimeType == "application/x-bittorrent" {
			return e.URL
		}
	}
	return ""
}

func (i Episode) ShowDirectoryName() string {
	s := i.ShowName
	s = strings.Trim(s, ".\\/")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	s = strings.ReplaceAll(s, ":", "-")
	return s
}

type FeedError string

func (fe FeedError) Error() string {
	return string(fe)
}

func ParseRSS(data []byte) (*Channel, error) {
	var rss rss

	if rss.Channel.TTL == 0 {
		rss.Channel.TTL = 15
	}

	if err := xml.Unmarshal(data, &rss); err != nil {
		return nil, err
	}
	if rss.Channel.Title == "" {
		return &rss.Channel, FeedError("channel has no title")
	}

	if len(rss.Channel.Episodes) == 0 {
		return &rss.Channel, FeedError("no episodes")
	}
	for k, v := range rss.Channel.Episodes {
		v.InfoHash = strings.ToLower(v.InfoHash)
		rss.Channel.Episodes[k] = v
	}
	return &rss.Channel, nil
}
