package pkg

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/contentadrstorage/pkg/domain"
)

type ContentAddressableStorage interface {
	Store(bucketId uint64, piece domain.Piece) domain.PieceUri
	Read(bucketId uint64, cid string) domain.Piece
	Search(query domain.Query) domain.SearchResult
}
