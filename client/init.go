package client

import (
	"context"
	"crypto/tls"
	"httpproxy/config"
	"net"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/op/go-logging"
)

var (
	cnfg     config.Client
	client   *HTTPProxyClient
	bogusdns *BogusDNS
	gfwlist  *GFWList
	enhttp   *EntryHTTPServer
	ensocks  *EntrySocksServer
	entproxy *EntryTproxyServer
	enredir  *EntryRedirectServer
)

var log = logging.MustGetLogger("HTTP Proxy")

// Initialize the client
func Initialize(c config.Client) {
	cnfg = c
	setLog()
	// Initialize Bogus DNS Server
	if cnfg.DNSListen != "" && runtime.GOOS == "linux" {
		prefix := net.ParseIP(cnfg.DNSPrefix)
		bogusdns = NewBogusDNS(cnfg.DNSListen, prefix, time.Duration(cnfg.DNSTTL)*time.Second)
		go bogusdns.ListenAndServe()
	}
	// Initialize Proxy client
	proxyURL, err := url.Parse(cnfg.Proxy)
	if err != nil {
		log.Fatal("Failed to parse proxy URL", err)
	}
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: cnfg.InsecureSkipVerify,
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
		ServerName: cnfg.Hostname,
	}
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
	// Update gfwlist
	if cnfg.GFWListURL != "" {
		gfwlist = NewGFWList()
		err := gfwlist.Update(cnfg.GFWListURL, client)
		if err != nil {
			log.Error(err)
		}
	}
	if bogusdns != nil && cnfg.DNSMasqCfg != "" {
		file, err := os.OpenFile(cnfg.DNSMasqCfg, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
		blacklist, _ := gfwlist.ExportDomains()
		if err := bogusdns.WriteDNSMasqConfig(file, blacklist); err != nil {
			log.Error(err)
		}
	}
	// Try to deploy
	if enredir != nil && bogusdns != nil {
		// Set the black and white list
		blacklist := []string{cnfg.DNSPrefix + "/16"}
		whitelist := make([]string, 0, 1)
		u, err := url.Parse(cnfg.Proxy)
		if err != nil {
			log.Error(err)
		} else {
			addr, err := net.ResolveTCPAddr("tcp", u.Host)
			if err != nil {
				log.Error(err)
			} else {
				whitelist = append(whitelist, addr.IP.String())
			}
		}
		if err := enredir.Deploy(blacklist, whitelist); err != nil {
			log.Error(err)
		}
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
	format = logging.MustStringFormatter("%{color}%{shortfunc:.12s}	▶ %{level:.4s} %{color:reset} %{message}")
	logging.SetFormatter(format)
	logging.SetLevel(level, "HTTP Proxy")
}
