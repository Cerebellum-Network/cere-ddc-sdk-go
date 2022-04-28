package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/pb"
)

type SearchResult struct {
	Pieces []*Piece
}

func (s *SearchResult) FromProto(pbSearchResult *pb.SearchResult) {
	s.Pieces = make([]*Piece, len(pbSearchResult.SignedPieces))

	for i, p := range pbSearchResult.SignedPieces {
		piece := &Piece{}
		piece.FromProto(p.Piece)

		s.Pieces[i] = piece
	}
}
