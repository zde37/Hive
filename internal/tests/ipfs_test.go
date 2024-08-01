package ipfs

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zde37/Hive/internal/ipfs"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		rpcAddr string
		wantErr bool
	}{
		{
			name:    "Valid RPC address",
			rpcAddr: "/ip4/127.0.0.1/tcp/5001",
			wantErr: false,
		},
		{
			name:    "Invalid RPC address",
			rpcAddr: "invalid_address",
			wantErr: true,
		},
		{
			name:    "Empty RPC address",
			rpcAddr: "",
			wantErr: true,
		},
		{
			name:    "IPv6 RPC address",
			rpcAddr: "/ip6/::1/tcp/5001",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := ipfs.NewClient(tt.rpcAddr)
			require.Equal(t, tt.wantErr, err != nil)
			require.Equal(t, tt.wantErr, client == nil)
		})
	}
}
