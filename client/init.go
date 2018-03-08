package client

import (
	"context"
	"crypto/tls"
	"httpproxy/config"
	"net"
	"net/url"
	"runtime"
	"time"

	"github.com/op/go-logging"
)

var (
	cnfg     config.Client
	client   *HTTPProxyClient
	bogusdns *BogusDNS
	enhttp   *EntryHTTPServer
	ensocks  *EntrySocksServer
	entproxy *EntryTproxyServer
	enredir  *EntryRedirectServer
)

var log = logging.MustGetLogger("HTTP Proxy")
var tlsConfig = &tls.Config{
	MinVersion:         tls.VersionTLS12,
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
	// Initialize Bogus DNS Server
	if cnfg.DNSListen != "" {
		prefix := net.ParseIP(cnfg.DNSPrefix)
		bogusdns = NewBogusDNS(cnfg.DNSListen, prefix, time.Duration(cnfg.DNSTTL)*time.Second)
		go bogusdns.ListenAndServe()
	}
	// Initialize Proxy client
	proxyURL, err := url.Parse(cnfg.Proxy)
	if err != nil {
		log.Fatal("Failed to parse proxy URL", err)
	}
	tlsConfig.InsecureSkipVerify = cnfg.InsecureSkipVerify
	tlsConfig.ServerName = cnfg.Hostname
	client = NewHTTPProxyClient(proxyURL, tlsConfig, bogusdns)
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
	// Cancel alll connection in client
	client.Cancel()
	// Shutdown the pool
	client.Pool.Release()
	return nil
}

//setLog() sets log output format.
func setLog() {
	var level logging.Level
	switch cnfg.LogLevel {
	case 0:
		level = logging.CRITICAL
		break
	default:
	case 1:
		level = logging.ERROR
		break
	case 2:
		level = logging.WARNING
		break
	case 3:
		level = logging.NOTICE
		break
	case 4:
		level = logging.INFO
		break
	case 5:
		level = logging.DEBUG
		break
	}

	var format logging.Formatter
	format = logging.MustStringFormatter("%{color}%{shortfunc}	▶ %{level:.4s} %{color:reset} %{message}")
	logging.SetFormatter(format)
	logging.SetLevel(level, "HTTP Proxy")
}
