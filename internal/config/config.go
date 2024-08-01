package config

// Config holds the configuration for the application.
type Config struct {
	RPC_ADDR     string
	WEB_UI_ADDR  string
	GATEWAY_ADDR string
	SERVER_ADDR  string
}

// Load creates a new Config struct with the provided configuration values. 
func Load(rpcAddr, webUIAddr, gatewayAddr, serverAddr string) *Config {
	return &Config{
		RPC_ADDR:     rpcAddr,
		WEB_UI_ADDR:  webUIAddr,
		GATEWAY_ADDR: gatewayAddr,
		SERVER_ADDR:  serverAddr,
	}
}
