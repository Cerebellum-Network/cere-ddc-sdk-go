package pkg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/cerebellum-network/cere-ddc-sdk-go/core/pkg/cid"
	"github.com/cerebellum-network/cere-ddc-sdk-go/core/pkg/crypto"
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/domain"
)

type ContentAddressableStorage interface {
	Store(ctx context.Context, piece *domain.Piece) (*PieceUri, error)
	Read(ctx context.Context, bucketId uint32, cid string) (*domain.Piece, error)
	Search(ctx context.Context, query *domain.Query) (*domain.SearchResult, error)
}

type contentAddressableStorage struct {
	scheme     crypto.Scheme
	cdnNodeUrl string
	cidBuilder *cid.Builder
	client     *http.Client
}

const basePath = "/api/rest/pieces"

func NewContentAddressableStorage(scheme crypto.Scheme, cdnNodeUrl string, cidBuilder *cid.Builder) ContentAddressableStorage {
	return &contentAddressableStorage{scheme: scheme, cdnNodeUrl: cdnNodeUrl, cidBuilder: cidBuilder, client: http.DefaultClient}
}

func (c *contentAddressableStorage) Store(ctx context.Context, piece *domain.Piece) (*PieceUri, error) {
	signPiece, pieceCid, err := c.signPiece(piece)
	if err != nil {
		return nil, err
	}
	body, err := signPiece.MarshalProto()
	if err != nil {
		return nil, err
	}

	_, err = c.sendRequest(ctx, "PUT", c.cdnNodeUrl+basePath, body, http.StatusCreated)
	if err != nil {
		return nil, fmt.Errorf("failed to store: %w", err)
	}

	pieceUri := &PieceUri{BucketId: piece.BucketId, Cid: pieceCid}
	return pieceUri, nil
}

func (c *contentAddressableStorage) Read(ctx context.Context, bucketId uint32, cid string) (*domain.Piece, error) {
	url := c.cdnNodeUrl + basePath + "/" + cid + "?bucketId=" + strconv.FormatUint(uint64(bucketId), 10)

	data, err := c.sendRequest(ctx, "GET", url, nil, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	signedPiece := &domain.SignedPiece{}
	err = signedPiece.UnmarshalProto(data)
	if err != nil {
		return nil, err
	}

	piece := signedPiece.Piece()
	return piece, nil
}

func (c *contentAddressableStorage) Search(ctx context.Context, query *domain.Query) (*domain.SearchResult, error) {
	body, err := query.MarshalProto()
	if err != nil {
		return nil, err
	}

	data, err := c.sendRequest(ctx, "GET", c.cdnNodeUrl+basePath, body, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	searchResult := &domain.SearchResult{}
	err = searchResult.UnmarshalProto(data)
	if err != nil {
		return nil, err
	}

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
		_, _ = io.Copy(ioutil.Discard, resp.Body)
		return nil, fmt.Errorf("fail status %s", resp.Status)
	}

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return result, err
}

func (c *contentAddressableStorage) signPiece(piece *domain.Piece) (*domain.SignedPiece, string, error) {
	pieceBytes, err := piece.MarshalProto()
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

	signedPiece := domain.NewSignedPiece(
		piece,
		pieceBytes,
		&domain.Signature{Value: signature, Signer: c.scheme.PublicKey(), Scheme: c.scheme.Name()},
	)

	return signedPiece, pieceCid, nil

}
