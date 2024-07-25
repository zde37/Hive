package config

type Config struct {
	RPC_ADDR     string
	WEB_UI_ADDR  string
	GATEWAY_ADDR string
}

func Load(rpcAddr, webUIAddr, gatewayAddr string) *Config {
	return &Config{
		RPC_ADDR:     rpcAddr,
		WEB_UI_ADDR:  webUIAddr,
		GATEWAY_ADDR: gatewayAddr,
	}
}
