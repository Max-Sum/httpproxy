package proxy

import (
	"os"
	"crypto/tls"
	"net"
	"net/url"
)

func NewProxyListener() net.Listener {
	var ln net.Listener
	listen, err := url.Parse(cnfg.Listen)
	if err != nil {
		log.Error("%v", err)
		return nil
	}
	q := listen.Query()
	if q.Get("tls") != "" {
		// Load Certificate
		cert, err := tls.LoadX509KeyPair(q.Get("cert"), q.Get("key"))
		if err != nil {
			log.Fatal(err)
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion: tls.VersionTLS12,
			PreferServerCipherSuites: true,
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
		switch listen.Scheme {
		case "unix":
			ln, err = tls.Listen("unix", listen.Host + listen.Path, config)
			if err1 := os.Chmod(listen.Host + listen.Path, 0666); err1 != nil {
				log.Fatal(err1)
			}
		default:
			ln, err = tls.Listen("tcp", listen.Host, config)
		}
	} else {
		switch listen.Scheme {
		case "unix":
			ln, err = net.Listen(listen.Scheme, listen.Host + listen.Path)
			if err1 := os.Chmod(listen.Host + listen.Path, 0666); err1 != nil {
				log.Fatal(err1)
			}
		default:
			ln, err = net.Listen("tcp", listen.Host)
		}
	}
	// Error
	if err != nil {
		log.Fatal("%v", err)
		return nil
	}
	return ln
}

func NewWebListener() net.Listener {
	var ln net.Listener
	listen, err := url.Parse(cnfg.WebListen)
	if err != nil {
		log.Error("%v", err)
		return nil
	}
	q := listen.Query()
	if q.Get("tls") != "" {
		// Load Certificate
		cert, err := tls.LoadX509KeyPair(q.Get("cert"), q.Get("key"))
		if err != nil {
			log.Fatal(err)
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion: tls.VersionTLS12,
			PreferServerCipherSuites: true,
			InsecureSkipVerify: true,
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
		switch listen.Scheme {
		case "unix":
			ln, err = tls.Listen(listen.Scheme, listen.Host + listen.Path, config)
			if err1 := os.Chmod(listen.Host + listen.Path, 0666); err1 != nil {
				log.Fatal(err1)
			}
		default:
			ln, err = tls.Listen("tcp", listen.Host, config)
		}
	} else {
		switch listen.Scheme {
		case "unix":
			ln, err = net.Listen(listen.Scheme, listen.Host + listen.Path)
			if err1 := os.Chmod(listen.Host + listen.Path, 0666); err1 != nil {
				log.Fatal(err1)
			}
		default:
			ln, err = net.Listen("tcp", listen.Host)
		}
	}
	// Error
	if err != nil {
		log.Fatal("%v", err)
		return nil
	}
	return ln
}