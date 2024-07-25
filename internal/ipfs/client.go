package ipfs

import (
	"context"
)

type Client interface {
	NodeID(ctx context.Context) (NodeInfo, error)
	Ping(ctx context.Context, peerID string) ([]PingInfo, error)
	Add(ctx context.Context, path string) (filePath, rootCid string, err error)
	GetObject(ctx context.Context, cid string, outputPath string) error
	GetConnectedPeers(ctx context.Context) ([]Peer, error)
	ListPins(ctx context.Context) ([]Pin, error)
	ListDir(ctx context.Context, dirPath string) ([]DirFileDetail, error)
	// FileLsRequest(ctx context.Context, path string) error
	PinObject(ctx context.Context, objectPath string) error
	UnPinObject(ctx context.Context, objectPath string) error
	DisplayFileContent(ctx context.Context, filePath string) (string, error)
}
