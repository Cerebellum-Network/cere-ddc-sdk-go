package domain

type (
	PieceUri struct {
		bucketId uint64
		cid      string
	}

	Piece struct {
		data []byte
		tags []Tag
	}

	Tag struct {
		key   string
		value string
	}

	Query struct {
		bucketId uint64
		tags     []Tag
	}

	SearchResult struct {
		pieces []Piece
	}
)
