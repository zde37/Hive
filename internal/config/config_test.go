package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		rpcAddr     string
		webUIAddr   string
		gatewayAddr string
		serverAddr  string
		want        *Config
	}{
		{
			name:        "All addresses provided",
			rpcAddr:     "localhost:8080",
			webUIAddr:   "localhost:8081",
			gatewayAddr: "localhost:8082",
			serverAddr:  "localhost:8083",
			want: &Config{
				RPC_ADDR:     "localhost:8080",
				WEB_UI_ADDR:  "localhost:8081",
				GATEWAY_ADDR: "localhost:8082",
				SERVER_ADDR:  "localhost:8083",
			},
		},
		{
			name:        "Empty addresses",
			rpcAddr:     "",
			webUIAddr:   "",
			gatewayAddr: "",
			serverAddr:  "",
			want: &Config{
				RPC_ADDR:     "",
				WEB_UI_ADDR:  "",
				GATEWAY_ADDR: "",
				SERVER_ADDR:  "",
			},
		},
		{
			name:        "Mixed empty and non-empty addresses",
			rpcAddr:     "localhost:8080",
			webUIAddr:   "",
			gatewayAddr: "localhost:8082",
			serverAddr:  "",
			want: &Config{
				RPC_ADDR:     "localhost:8080",
				WEB_UI_ADDR:  "",
				GATEWAY_ADDR: "localhost:8082",
				SERVER_ADDR:  "",
			},
		},
		{
			name:        "IPv6 addresses",
			rpcAddr:     "[::1]:8080",
			webUIAddr:   "[::1]:8081",
			gatewayAddr: "[::1]:8082",
			serverAddr:  "[::1]:8083",
			want: &Config{
				RPC_ADDR:     "[::1]:8080",
				WEB_UI_ADDR:  "[::1]:8081",
				GATEWAY_ADDR: "[::1]:8082",
				SERVER_ADDR:  "[::1]:8083",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Load(tt.rpcAddr, tt.webUIAddr, tt.gatewayAddr, tt.serverAddr)
			require.Equal(t, tt.want, got)
		})
	}
}
