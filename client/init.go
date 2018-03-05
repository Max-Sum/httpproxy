package client

import (
	"context"
	"net/url"
	"crypto/tls"
	"runtime"
	"github.com/op/go-logging"
	"httpproxy/config"
)

var (
	cnfg     config.Client
	client   *HTTPProxyClient
	enhttp   *EntryHTTPServer
	ensocks  *EntrySocksServer
	entproxy *EntryTproxyServer
	enredir  *EntryRedirectServer
)

var log = logging.MustGetLogger("HTTP Proxy")
var tlsConfig = &tls.Config{
	MinVersion: tls.VersionTLS12,
	InsecureSkipVerify: false,
	ClientSessionCache: tls.NewLRUClientSessionCache(128),
	CipherSuites: []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	},
}

// Initialize the client
func Initialize(c config.Client) {
	cnfg = c
	setLog()
	// Initialize Proxy client
	proxyURL, err := url.Parse(cnfg.Proxy)
	if err != nil {
		log.Fatal("Failed to parse proxy URL", err)
	}
	tlsConfig.InsecureSkipVerify = cnfg.InsecureSkipVerify
	tlsConfig.ServerName = proxyURL.Hostname()
	client = NewHTTPProxyClient(proxyURL, tlsConfig)
	client.SetBasicAuth(cnfg.Username, cnfg.Password)
	// Initialize Entrypoints
	if cnfg.HTTPListen != "" {
		enhttp = NewEntryHTTPServer(cnfg.HTTPListen, client)
		go enhttp.ListenAndServe()
	}
	if cnfg.SocksListen != "" {
		ensocks = NewEntrySocksServer(cnfg.SocksListen, client)
		go ensocks.ListenAndServe()
	}
	if cnfg.RedirListen != "" && runtime.GOOS == "linux" {
		enredir = NewEntryRedirectServer(cnfg.RedirListen, client)
		go enredir.ListenAndServe()
	}
	if cnfg.TProxyListen != "" && runtime.GOOS == "linux" {
		entproxy = NewEntryTProxyServer(cnfg.TProxyListen, client)
		go entproxy.ListenAndServe()
	}
}

func Shutdown() error {
	var err error
	if enhttp != nil {
		err = enhttp.Shutdown(context.Background())
		if err != nil {
			return err
		}
	}
	if ensocks != nil {
		err = ensocks.Shutdown()
		if err != nil {
			return err
		}
	}
	if enredir != nil {
		err = enredir.Shutdown()
		if err != nil {
			return err
		}
	}
	if entproxy != nil {
		err = entproxy.Shutdown()
		if err != nil {
			return err
		}
	}
	return nil
}

// Close all service forcefully
func Close() error {
	var err error
	if enhttp != nil {
		err = enhttp.Close()
		if err != nil {
			return err
		}
	}
	// Shutdown other platform
	if ensocks != nil {
		err = ensocks.Shutdown()
		if err != nil {
			return err
		}
	}
	if enredir != nil {
		err = enredir.Shutdown()
		if err != nil {
			return err
		}
	}
	if entproxy != nil {
		err = entproxy.Shutdown()
		if err != nil {
			return err
		}
	}
	client.Cancel()
	return nil
}

//setLog() sets log output format.
func setLog() {
	var level logging.Level
	if cnfg.LogLevel == 1 {
		level = logging.DEBUG
	} else {
		level = logging.INFO
	}

	var format logging.Formatter
	if level == logging.DEBUG {
		format = logging.MustStringFormatter("%{shortfile} %{level} %{message}")
	} else {
		format = logging.MustStringFormatter("%{level} %{message}")
	}
	logging.SetFormatter(format)
	logging.SetLevel(level, "HTTP Proxy")
}