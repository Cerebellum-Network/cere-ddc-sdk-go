// Package storage stores and reads content on a DDC storage node over its gRPC
// API.
//
// DDC storage is content-addressed. A raw piece is a blob of up to
// MaxRawPieceSize bytes; a DAG node holds data and named links to pieces and
// other nodes; a CNS name is a mutable, signed pointer from a name in a bucket
// to a CID. Every call carries a DDC access token for the bucket (see the auth
// module): PUT for writes, GET for reads.
//
// Client covers the common write-and-resolve path. The generated gRPC clients
// and servers in fileapi, dagapi and cnsapi cover the full API; storagetest
// runs an in-process storage node for tests.
package storage
