package domain

import (
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

type SearchResult struct {
	SearchedPieces []*SearchedPiece
}

var _ Protobufable = (*SearchResult)(nil)

func (sr *SearchResult) ToProto() *pb.SearchResult {
	pbSearchResult := &pb.SearchResult{
		SearchedPieces: make([]*pb.SearchedPiece, len(sr.SearchedPieces)),
	}

	for i, searchedPiece := range sr.SearchedPieces {
		pbSearchResult.SearchedPieces[i] = searchedPiece.ToProto()
	}

	return pbSearchResult
}

func (sr *SearchResult) ToDomain(pbSearchResult *pb.SearchResult) {
	sr.SearchedPieces = make([]*SearchedPiece, len(pbSearchResult.SearchedPieces))

	for i, pbSearchedPiece := range pbSearchResult.SearchedPieces {
		searchedPiece := &SearchedPiece{}
		searchedPiece.ToDomain(pbSearchedPiece)
		sr.SearchedPieces[i] = searchedPiece
	}
}

func (sr *SearchResult) MarshalProto() ([]byte, error) {
	return proto.Marshal(sr.ToProto())
}

func (sr *SearchResult) UnmarshalProto(searchResultAsBytes []byte) error {
	pbSearchResult := &pb.SearchResult{}
	if err := proto.Unmarshal(searchResultAsBytes, pbSearchResult); err != nil {
		return err
	}

	sr.ToDomain(pbSearchResult)
	return nil
}

type SearchedPiece struct {
	SignedPiece *SignedPiece
	Cid         string
}

var _ Protobufable = (*SearchedPiece)(nil)

func (sp *SearchedPiece) ToProto() *pb.SearchedPiece {
	return &pb.SearchedPiece{
		Cid:         sp.Cid,
		SignedPiece: sp.SignedPiece.ToProto(),
	}
}

func (sp *SearchedPiece) ToDomain(pbSearchPiece *pb.SearchedPiece) {
	sp.Cid = pbSearchPiece.Cid
	sp.SignedPiece = &SignedPiece{}

	sp.SignedPiece.ToDomain(pbSearchPiece.SignedPiece)
}

func (sp *SearchedPiece) MarshalProto() ([]byte, error) {
	return proto.Marshal(sp.ToProto())
}

func (sp *SearchedPiece) UnmarshalProto(searchedPieceAsBytes []byte) error {
	pbSearchedPiece := &pb.SearchedPiece{}
	if err := proto.Unmarshal(searchedPieceAsBytes, pbSearchedPiece); err != nil {
		return err
	}

	sp.ToDomain(pbSearchedPiece)
	return nil
}
