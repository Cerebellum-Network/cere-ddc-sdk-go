package storage

import (
	"bytes"
	"context"
	"fmt"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contentadrstorage/pkg/domain"
	"github.com/cerebellum-network/cere-ddc-sdk-go/pb"
	"github.com/cerebellum-network/cere-ddc-sdk-go/pkg/cid"
	"github.com/cerebellum-network/cere-ddc-sdk-go/pkg/crypto"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"strconv"
)

type ContentAddressableStorage interface {
	Store(ctx context.Context, piece *domain.Piece) (*domain.PieceUri, error)
	Read(ctx context.Context, bucketId uint32, cid string) (*domain.Piece, error)
	Search(ctx context.Context, query *domain.Query) (*domain.SearchResult, error)
}

type contentAddressableStorage struct {
	scheme         crypto.Scheme
	gatewayNodeUrl string
	cidBuilder     *cid.Builder
	client         *http.Client
}

const basePath = "/api/rest/pieces"

func NewContentAddressableStorage(scheme crypto.Scheme, gatewayNodeUrl string) ContentAddressableStorage {
	return &contentAddressableStorage{scheme: scheme, gatewayNodeUrl: gatewayNodeUrl, cidBuilder: cid.DefaultBuilder(), client: http.DefaultClient}
}

func (c *contentAddressableStorage) Store(ctx context.Context, piece *domain.Piece) (*domain.PieceUri, error) {
	pbPiece := piece.ToProto()

	signPiece, pieceCid, err := c.signPiece(pbPiece)
	if err != nil {
		return nil, err
	}
	body, err := proto.Marshal(signPiece)
	if err != nil {
		return nil, err
	}

	_, err = c.sendRequest(ctx, "PUT", c.gatewayNodeUrl+basePath, body, http.StatusCreated)
	if err != nil {
		return nil, fmt.Errorf("failed to store: %w", err)
	}

	pieceUri := &domain.PieceUri{BucketId: piece.BucketId, Cid: pieceCid}
	return pieceUri, nil
}

func (c *contentAddressableStorage) Read(ctx context.Context, bucketId uint32, cid string) (*domain.Piece, error) {
	url := c.gatewayNodeUrl + basePath + "/" + cid + "?bucketId=" + strconv.FormatUint(uint64(bucketId), 10)

	data, err := c.sendRequest(ctx, "GET", url, nil, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	pbSignedPiece := &pb.SignedPiece{}
	err = proto.Unmarshal(data, pbSignedPiece)
	if err != nil {
		return nil, err
	}

	piece := &domain.Piece{}
	piece.FromProto(pbSignedPiece.Piece)

	return piece, nil
}

func (c *contentAddressableStorage) Search(ctx context.Context, query *domain.Query) (*domain.SearchResult, error) {
	pbQuery := query.ToProto()
	body, err := proto.Marshal(pbQuery)
	if err != nil {
		return nil, err
	}

	data, err := c.sendRequest(ctx, "GET", c.gatewayNodeUrl+basePath, body, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	pbSearchResult := &pb.SearchResult{}
	err = proto.Unmarshal(data, pbSearchResult)
	if err != nil {
		return nil, err
	}

	searchResult := &domain.SearchResult{}
	searchResult.FromProto(pbSearchResult)

	return searchResult, nil
}

func (c *contentAddressableStorage) sendRequest(ctx context.Context, method string, url string, body []byte, expectedStatus int) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		//ToDO we need to read body
		return nil, fmt.Errorf("fail status %s", resp.Status)
	}

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return result, err
}

func (c *contentAddressableStorage) signPiece(piece *pb.Piece) (*pb.SignedPiece, string, error) {
	pieceBytes, err := proto.Marshal(piece)
	if err != nil {
		return nil, "", fmt.Errorf("failed marshal piece proto: %w", err)
	}

	pieceCid, err := c.cidBuilder.Build(pieceBytes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to build CID: %w", err)
	}

	signature, err := c.scheme.Sign([]byte(pieceCid))
	if err != nil {
		return nil, "", fmt.Errorf("failed to sign piece: %w", err)
	}

	signedPiece := &pb.SignedPiece{
		Piece:     piece,
		Signature: &pb.Signature{Value: signature, Signer: c.scheme.PublicKey(), Scheme: c.scheme.Name()},
	}

	return signedPiece, pieceCid, nil

}
