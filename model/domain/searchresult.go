package domain

import (
	domain "github.com/cerebellum-network/cere-ddc-sdk-go/model"
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	"google.golang.org/protobuf/proto"
)

type SearchResult struct {
	SignedPieces []*SignedPiece
}

var _ domain.Protobufable = (*SearchResult)(nil)

func (sr *SearchResult) ToProto() *pb.SearchResult {
	pbSearchResult := &pb.SearchResult{
		SignedPieces: make([]*pb.SignedPiece, len(sr.SignedPieces)),
	}

	for i, signedPiece := range sr.SignedPieces {
		pbSearchResult.SignedPieces[i] = signedPiece.ToProto()
	}

	return pbSearchResult
}

func (sr *SearchResult) ToDomain(pbSearchResult *pb.SearchResult) {
	sr.SignedPieces = make([]*SignedPiece, len(pbSearchResult.SignedPieces))

	for i, pbSignedPiece := range pbSearchResult.SignedPieces {
		signedPiece := &SignedPiece{}
		signedPiece.ToDomain(pbSignedPiece)
		sr.SignedPieces[i] = signedPiece
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
