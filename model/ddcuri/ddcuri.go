package ddcuri

import (
	"fmt"
	"strings"
)

type DdcQuery struct {
	Organization string

	BucketName  string
	BucketId    uint32
	BucketIdSet bool

	Protocol string
	Cid      string
	Path     []string

	Options string
}

func Parse(uri string) (q DdcQuery, err error) {
	uri = consumeOptions(&q, uri)
	parts := splitParts(uri)
	err = consumeMain(&q, parts)
	return q, err
}

func ParseWebUrl(url string) (q DdcQuery, err error) {
	position := strings.Index(url, "/ddc/")
	if position == -1 {
		return q, fmt.Errorf("not a DDC URL (%s)", url)
	}
	uri := url[position:]
	return Parse(uri)
}

func (q *DdcQuery) String() string {
	return q.toUri()
}

const (
	DDC    = "ddc"
	ORG    = "org"
	BUC    = "buc"
	IPIECE = "ipiece"
	IFILE  = "ifile"
	PIECE  = "piece"
	FILE   = "file"
)
