package ipfs

import (
	"fmt"

	"github.com/ipfs/kubo/client/rpc"
	"github.com/multiformats/go-multiaddr"
)

// NewClient creates a new IPFS RPC client using the provided RPC address. 
func NewClient(rpcAddr string) (*rpc.HttpApi, error) {
	addr, err := multiaddr.NewMultiaddr(rpcAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create multiaddr: %v", err)
	}

	rpc, err := rpc.NewApi(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create rpc api: %v", err)
	}

	return rpc, nil
}
