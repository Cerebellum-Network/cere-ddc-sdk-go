package ddcuri

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
