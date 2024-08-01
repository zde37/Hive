package ipfs

import (
	"context"
)

type Client interface {
	NodeInfo(ctx context.Context, peerID string) (NodeInfo, error)
	Ping(ctx context.Context, peerID string) ([]PingInfo, error)
	Add(ctx context.Context, fileName, filePath string) (string, string, error)
	DownloadFile(ctx context.Context, cid string) ([]byte, error)
	GetConnectedPeers(ctx context.Context) ([]Peer, error)
	ListPins(ctx context.Context) (any, error)
	PinObject(ctx context.Context, name, objectPath string) error
	DeleteFile(ctx context.Context, objectPath string) error
	DisplayFileContent(ctx context.Context, filePath string) (string, error)
	DownloadDir(ctx context.Context, cid string, outputPath string) error
	ListDir(ctx context.Context, dirPath string) ([]DirFileDetail, error) 
}
