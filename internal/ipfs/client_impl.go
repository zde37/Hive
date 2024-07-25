package ipfs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/client/rpc"
	iface "github.com/ipfs/kubo/core/coreiface"
	"github.com/ipfs/kubo/core/coreiface/options"
)

type ClientImpl struct {
	rpc    *rpc.HttpApi
	rpcUrl string
}

type NodeInfo struct {
	Addresses    []string
	AgentVersion string
	ID           string
	Protocols    []string
	PublicKey    string
}

type Peer struct {
	ID        string
	Address   string
	Direction string
	Latency   time.Duration
}

type PingInfo struct {
	Success bool
	Text    string
	Time    time.Duration
}

type Pin struct {
	Name    string
	Path    string
	RootCid string
	Type    string
}

type DirFileDetail struct {
	Name string
	Cid  cid.Cid

	Size uint64         // The size of the file in bytes (or the size of the symlink).
	Type iface.FileType // The type of the file.
}

func NewClientImpl(rpc *rpc.HttpApi, url string) Client {
	return &ClientImpl{
		rpc:    rpc,
		rpcUrl: url,
	}
}

func (c *ClientImpl) Add(ctx context.Context, path string) (filePath, rootCid string, err error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", "", err
	}

	var node files.Node
	if stat.IsDir() {
		node, err = files.NewSerialFile(path, false, stat)
		if err != nil {
			return "", "", err
		}
	} else {
		file, err := os.Open(path)
		if err != nil {
			return "", "", err
		}
		defer file.Close()
		node = files.NewReaderStatFile(file, stat)
	}

	opts := []options.UnixfsAddOption{
		options.Unixfs.Pin(true),
		options.Unixfs.CidVersion(1),
	}

	immutPath, err := c.rpc.Unixfs().Add(ctx, node, opts...)
	if err != nil {
		return "", "", err
	}

	return immutPath.String(), immutPath.RootCid().String(), nil
}

// func (c *ClientImpl) FileLsRequest(ctx context.Context, path string) error {
// 	if path == "" {
// 		path = "/files"
// 	}
// 	return c.rpc.Request("files/ls").
// 		Arguments(path).
// 		Exec(ctx, nil)
// }

// func (c *ClientImpl) fileCpRequest(ctx context.Context, srcPath string) error {
// 	return c.rpc.Request("files/cp").
// 		Arguments(srcPath, "/files").
// 		Exec(ctx, nil)
// }

func (c *ClientImpl) NodeID(ctx context.Context) (NodeInfo, error) {
	var res NodeInfo
	err := c.rpc.Request("id").
		Exec(ctx, &res)
	return res, err
}

func (c *ClientImpl) DisplayFileContent(ctx context.Context, filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("no file path provided")
	}

	res, err := c.rpc.Request("cat").
		Arguments(filePath).
		Send(ctx)
	if err != nil {
		return "", err
	}
	defer res.Output.Close()

	var builder strings.Builder
	_, err = io.Copy(&builder, res.Output)
	if err != nil {
		return "", err
	}

	return builder.String(), nil
}

func (c *ClientImpl) Ping(ctx context.Context, peerID string) ([]PingInfo, error) {
	response, err := c.rpc.Request("ping").
		Arguments(peerID).
		Send(ctx) // Exec() does not decode the ping response well, fix it and create a pull request
	if err != nil {
		return nil, err
	}
	defer response.Output.Close()

	var res []PingInfo
	decoder := json.NewDecoder(response.Output)
	for {
		var r PingInfo
		if err := decoder.Decode(&r); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (c *ClientImpl) PinObject(ctx context.Context, objectPath string) error {
	rootPath, err := path.NewPath(objectPath)
	if err != nil {
		return err
	}
	return c.rpc.Pin().Add(ctx, rootPath)
}

func (c *ClientImpl) UnPinObject(ctx context.Context, objectPath string) error {
	rootPath, err := path.NewPath(objectPath)
	if err != nil {
		return err
	}

	res, isPinned, err := c.rpc.Pin().IsPinned(ctx, rootPath)
	if err != nil {
		return err
	}
	if !isPinned {
		return fmt.Errorf("object is not pinned")
	}
	if strings.Contains(res, "indirect through") { // object pinned indirectly
		return fmt.Errorf("%s", res)
	}

	return c.rpc.Pin().Rm(ctx, rootPath)
}

func (c *ClientImpl) GetObject(ctx context.Context, cid string, outputPath string) error {
	path, err := getPathFromCid(cid)
	if err != nil {
		return err
	}

	node, err := c.rpc.Unixfs().Get(ctx, path)
	if err != nil {
		return err
	}

	switch n := node.(type) {
	case files.File:
		return writeFile(n, outputPath)
	case files.Directory:
		return writeDirectory(n, outputPath)
	default:
		return fmt.Errorf("unsupported node type")
	}
}

func (c *ClientImpl) GetConnectedPeers(ctx context.Context) ([]Peer, error) {
	connectedPeers, err := c.rpc.Swarm().Peers(ctx)
	if err != nil {
		return nil, err
	}

	peers := make([]Peer, len(connectedPeers))
	for _, peer := range connectedPeers {
		latency, err := peer.Latency()
		if err != nil {
			return nil, err
		}

		peers = append(peers, Peer{
			ID:        peer.ID().String(),
			Address:   peer.Address().String(),
			Direction: peer.Direction().String(),
			Latency:   latency,
		})
	}

	return peers, nil
}

func getPathFromCid(cidString string) (path.Path, error) {
	c, err := cid.Decode(cidString)
	if err != nil {
		return nil, err
	}
	return path.FromCid(c), nil
}

func (c *ClientImpl) ListPins(ctx context.Context) ([]Pin, error) {
	files := []Pin{}
	pinsChan, err := c.rpc.Pin().Ls(ctx)
	if err != nil {
		return nil, err
	}

	for pin := range pinsChan {
		files = append(files, Pin{
			Name:    pin.Name(),
			RootCid: pin.Path().RootCid().String(),
			Path:    pin.Path().String(),
			Type:    pin.Type(),
		})
	}
	return files, nil
}

func (c *ClientImpl) ListDir(ctx context.Context, dirPath string) ([]DirFileDetail, error) {
	if dirPath == "" {
		return nil, fmt.Errorf("no directory path provided")
	}
	rootPath, err := path.NewPath(dirPath)
	if err != nil {
		return nil, err
	}

	files := []DirFileDetail{}
	entries, err := c.rpc.Unixfs().Ls(ctx, rootPath)
	if err != nil {
		return nil, err
	}

	for entry := range entries {
		files = append(files, DirFileDetail{
			Name: entry.Name,
			Cid:  entry.Cid,
			Size: entry.Size,
			Type: entry.Type,
		})
	}
	return files, nil
}

func writeFile(file files.File, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, file)
	return err
}

func writeDirectory(dir files.Directory, path string) error {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}

	entries := dir.Entries()
	for entries.Next() {
		node := entries.Node()
		childPath := filepath.Join(path, entries.Name())

		switch n := node.(type) {
		case files.File:
			err = writeFile(n, childPath)
		case files.Directory:
			err = writeDirectory(n, childPath)
		default:
			err = fmt.Errorf("unsupported node type")
		}

		if err != nil {
			return err
		}
	}

	return entries.Err()
}
