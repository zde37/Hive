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

// ClientImpl is the implementation of the IPFS client.
type ClientImpl struct {
	rpc *rpc.HttpApi // the RPC client.
}

// NodeInfo contains information about an IPFS node. TODO: look into 'AgentVersion' and 'PublicKey' fields
type NodeInfo struct {
	Addresses    []string `json:"addresses"`    // the addresses of the node.
	AgentVersion string   `json:"AgentVersion"` // the version of the IPFS agent.
	ID           string   `json:"id"`           // the ID of the node.
	Protocols    []string `json:"protocols"`    // the protocols supported by the node.
	PublicKey    string   `json:"PublicKey"`    // the public key of the node.
}

// Node represents an IPFS node.
type Node struct {
	ID        string `json:"id"`        // the ID of the node.
	Address   string `json:"address"`   // the address of the node.
	Direction string `json:"direction"` // the direction of the connection (inbound or outbound).
	Latency   int64  `json:"latency"`   // the latency of the connection to the node.
}

// PingInfo contains the result of an IPFS ping operation.
type PingInfo struct {
	Success bool          `json:"success"` // whether the ping was successful.
	Text    string        `json:"text"`    // the text output of the ping.
	Time    time.Duration `json:"time"`    // the duration of the ping.
}

// Pin represents a pinned IPFS object.
// type Pin struct {
// 	Name    string `json:"name"`     // the name of the pinned object.
// 	Path    string `json:"path"`     // the path of the pinned object.
// 	RootCid string `json:"root_cid"` // the root CID of the pinned object.
// 	Type    string `json:"type"`     // the type of the pinned object.
// }

// DirFileDetail represents details about a file or directory in IPFS.
type DirFileDetail struct {
	Name string  `json:"name"` // the name of the file or directory.
	Cid  cid.Cid `json:"cid"`  // the CID of the file or directory.

	Size uint64         `json:"size"` // the size of the file in bytes (or the size of the symlink).
	Type iface.FileType `json:"type"` // the type of the file.
}

// NewClientImpl creates a new IPFS client implementation.
func NewClientImpl(rpc *rpc.HttpApi) Client {
	return &ClientImpl{
		rpc: rpc,
	}
}

// Add adds a file or directory to IPFS and returns the immutable path and root CID of the added object.
func (c *ClientImpl) Add(ctx context.Context, fileName, filePath string) (string, string, error) {
	if fileName == "" || filePath == "" {
		return "", "", fmt.Errorf("file name and path are required")
	}
	stat, err := os.Stat(filePath)
	if err != nil {
		return "", "", err
	}

	var node files.Node
	if stat.IsDir() {
		node, err = files.NewSerialFile(filePath, false, stat)
		if err != nil {
			return "", "", err
		}
	} else {
		file, err := os.Open(filePath)
		if err != nil {
			return "", "", err
		}
		defer file.Close()
		node = files.NewReaderStatFile(file, stat)
	}

	opts := []options.UnixfsAddOption{
		options.Unixfs.Pin(false),
		options.Unixfs.CidVersion(1),
	}

	// add object to ipfs node
	immutPath, err := c.rpc.Unixfs().Add(ctx, node, opts...)
	if err != nil {
		return "", "", err
	}

	p, err := c.getPathFromCid(immutPath.RootCid().String())
	if err != nil {
		return "", "", err
	}

	// pin the object
	if err = c.PinObject(ctx, fileName, p.String()); err != nil {
		return "", "", err
	}

	return immutPath.String(), immutPath.RootCid().String(), nil
}

// NodeInfo returns information about the local IPFS node, including its addresses, agent version, ID, supported protocols, and public key.
func (c *ClientImpl) NodeInfo(ctx context.Context, peerID string) (NodeInfo, error) {
	if peerID == "" {
		return NodeInfo{}, fmt.Errorf("peer id is required")
	}
	var res NodeInfo
	err := c.rpc.Request("id").
		Arguments(peerID).
		Exec(ctx, &res)
	return res, err
}

// DisplayFileContent returns the contents of the file at the given path as a string. If the path is empty, it returns an error.
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

	if res.Output == nil {
		return "", fmt.Errorf("no output from cat request or failed to read folder")
	}
	defer res.Output.Close()

	var builder strings.Builder
	_, err = io.Copy(&builder, res.Output)
	if err != nil {
		return "", err
	}

	return builder.String(), nil
}

// Ping sends a ping request to the IPFS peer with the given ID and returns the ping response, which includes information about the success, text, and duration of the ping.
func (c *ClientImpl) Ping(ctx context.Context, peerID string) ([]PingInfo, error) {
	if peerID == "" {
		return nil, fmt.Errorf("no peer id provided")
	}

	response, err := c.rpc.Request("ping").
		Arguments(peerID).
		Send(ctx) // Exec() does not decode the ping response well, fix it and create a pull request
	if err != nil {
		return nil, err
	}

	if response.Output == nil {
		return nil, fmt.Errorf("no output from ping request or failed to ping self")
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

// PinObject pins the IPFS object at the given path, ensuring that it is not garbage collected.
func (c *ClientImpl) PinObject(ctx context.Context, name, objectPath string) error {
	rootPath, err := path.NewPath(objectPath)
	if err != nil {
		return err
	}

	return c.rpc.Request("pin/add").
		Arguments(rootPath.String()).
		Option("name", name).
		Exec(ctx, nil)
}

// garbageCollection performs a garbage collection on the IPFS repository to remove any unpinned objects.
func (c *ClientImpl) garbageCollection(ctx context.Context) error {
	return c.rpc.Request("repo/gc").
		Exec(ctx, nil)
}

// DeleteFile unpins the IPFS object at the given path and then performs a garbage collection to remove the unpinned object.
func (c *ClientImpl) DeleteFile(ctx context.Context, objectPath string) error {
	rootPath, err := path.NewPath(objectPath)
	if err != nil {
		return err
	}

	res, isPinned, err := c.rpc.Pin().IsPinned(ctx, rootPath)
	if err != nil {
		return err
	}
	if !isPinned {
		return fmt.Errorf("..object is not pinned")
	}
	if strings.Contains(res, "indirect through") { // object pinned indirectly
		return fmt.Errorf("..object is pinned %s", res)
	}

	// first unpin object, then perform garbage collection
	err = c.rpc.Pin().Rm(ctx, rootPath)
	if err != nil {
		return err
	}

	return c.garbageCollection(ctx)
}

// DownloadFile downloads the IPFS object with the given CID and returns its contents as a byte slice.
// If the object is not a file, an error is returned.
func (c *ClientImpl) DownloadFile(ctx context.Context, cid string) ([]byte, error) {
	path, err := path.NewPath("/ipfs/" + cid)
	if err != nil {
		return nil, err
	}

	node, err := c.rpc.Unixfs().Get(ctx, path)
	if err != nil {
		return nil, err
	}

	file, ok := node.(files.File)
	if !ok {
		return nil, fmt.Errorf("not a file")
	}

	return io.ReadAll(file)
}

// DownloadDir retrieves the IPFS object (directory) at the given CID and writes it to the specified output path.
func (c *ClientImpl) DownloadDir(ctx context.Context, cid string, outputPath string) error {
	path, err := c.getPathFromCid(cid)
	if err != nil {
		return err
	}

	node, err := c.rpc.Unixfs().Get(ctx, path)
	if err != nil {
		return err
	}
	
	dir, ok := node.(files.Directory)
    if !ok {
        return fmt.Errorf("node is not a directory")
    }
	return writeDirectory(dir, outputPath)
}

// ListConnectedNodes returns a list of all the nodes that the current IPFS node is connected to.
// For each node, the function returns the node ID, address, connection direction, and latency.
func (c *ClientImpl) ListConnectedNodes(ctx context.Context) ([]Node, error) {
	connectedNodes, err := c.rpc.Swarm().Peers(ctx)
	if err != nil {
		return nil, err
	}

	var nodes []Node
	for _, node := range connectedNodes {
		latency, err := node.Latency()
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, Node{
			ID:        node.ID().String(),
			Address:   node.Address().String(),
			Direction: node.Direction().String(),
			Latency:   latency.Milliseconds(),
		})
	}

	return nodes, nil
}

// getPathFromCid converts a CID string to a path.Path.
func (c *ClientImpl) getPathFromCid(cidString string) (path.Path, error) {
	cid, err := cid.Decode(cidString)
	if err != nil {
		return nil, err
	}
	return path.FromCid(cid), nil
}

// ListPins returns a list of all the IPFS objects that are currently pinned.
func (c *ClientImpl) ListPins(ctx context.Context) (any, error) { 
	response, err := c.rpc.Request("pin/ls").
		Option("names", true).
		Send(ctx)
	if err != nil {
		return nil, err
	}
	if response.Output == nil {
		return nil, fmt.Errorf("no output from list pins request")
	}
	defer response.Output.Close()

	var res any // TODO: create and use a custom type 
	if err = json.NewDecoder(response.Output).Decode(&res); err != nil {
		return nil, err
	}

	return res, err
}

// ListDir returns a list of all the files and directories in the specified directory path.
// For each file/directory, the function returns the name, CID, size, and type.
func (c *ClientImpl) ListDir(ctx context.Context, dirPath string) ([]DirFileDetail, error) {
	if dirPath == "" {
		return nil, fmt.Errorf("no directory path provided")
	}
	rootPath, err := path.NewPath(dirPath)
	if err != nil {
		return nil, err
	}

	var files []DirFileDetail
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

// writeFile writes the contents of the specified files.File to the specified file path.
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
            err = fmt.Errorf("unsupported node type: %T", n)
        }

        if err != nil {
            return err
        }
    }

    return entries.Err()
}
