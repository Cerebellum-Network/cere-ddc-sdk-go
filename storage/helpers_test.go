package storage_test

import (
	"context"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"github.com/cerebellum-network/cere-ddc-sdk-go/storage/cid"
	"github.com/cerebellum-network/cere-ddc-sdk-go/storage/cnsapi"
	"github.com/cerebellum-network/cere-ddc-sdk-go/storage/fileapi"
)

// serve runs register's services on an in-memory listener and returns a
// connection to them.
func serve(t *testing.T, register func(*grpc.Server)) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	register(srv)
	go func() { _ = srv.Serve(lis) }()
	conn, err := grpc.NewClient("passthrough:///capture",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
		srv.Stop()
	})
	return conn
}

func tokenOf(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	if v := md.Get("token"); len(v) > 0 {
		return v[0]
	}
	return ""
}

func sumCID(t *testing.T, data []byte) []byte {
	t.Helper()
	c, err := cid.Sum(cid.DefaultParams(cid.ContentTypeDagPb), cid.MhTypeBlake3, data)
	require.NoError(t, err)
	return c.Bytes()
}

// capturingFile records every PutRawPiece message and the token it came with.
type capturingFile struct {
	fileapi.UnimplementedFileApiServer
	mu       sync.Mutex
	messages []*fileapi.PutRawPieceRequest
	tokens   []string
}

func (c *capturingFile) serve(t *testing.T) *grpc.ClientConn {
	return serve(t, func(s *grpc.Server) { fileapi.RegisterFileApiServer(s, c) })
}

func (c *capturingFile) PutRawPiece(stream fileapi.FileApi_PutRawPieceServer) error {
	c.mu.Lock()
	c.tokens = append(c.tokens, tokenOf(stream.Context()))
	c.messages = nil
	c.mu.Unlock()
	var data []byte
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		c.mu.Lock()
		c.messages = append(c.messages, msg)
		c.mu.Unlock()
		data = append(data, msg.GetContent().GetData()...)
	}
	return stream.SendAndClose(&fileapi.PutRawPieceResponse{Cid: sumCIDNoT(data)})
}

func sumCIDNoT(data []byte) []byte {
	c, err := cid.Sum(cid.DefaultParams(cid.ContentTypeRaw), cid.MhTypeBlake3, data)
	if err != nil {
		panic(err)
	}
	return c.Bytes()
}

// capturingCNS records the CNS put request and the token it came with.
type capturingCNS struct {
	cnsapi.UnimplementedCnsApiServer
	put    *cnsapi.PutRequest
	tokens []string
}

func (c *capturingCNS) serve(t *testing.T) *grpc.ClientConn {
	return serve(t, func(s *grpc.Server) { cnsapi.RegisterCnsApiServer(s, c) })
}

func (c *capturingCNS) Put(ctx context.Context, req *cnsapi.PutRequest) (*cnsapi.PutResponse, error) {
	c.put = req
	c.tokens = append(c.tokens, tokenOf(ctx))
	return &cnsapi.PutResponse{}, nil
}
