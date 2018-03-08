package client

import (
	"fmt"
	"hash/fnv"
	"net"
	"time"

	"github.com/miekg/dns"
)

// BogusDNS is a fake DNS server.
// It translate any domain into fake private IP,
// which could then be translate back into domain.
// This can cooperate with Redir or TProxy to send
// domain to remote proxy server, avoiding a DNS resolving
// over the proxy.
type BogusDNS struct {
	srv         dns.Server
	requestChan chan *requestItem
	IPPrefix    net.IP
	DNSTTL      time.Duration // TTL to reply the request
	TTL         time.Duration // TTL to stay in the map
	IPIndex     [65536]*bogusItem
}

type bogusItem struct {
	domain string
	ctime  time.Time // item create time
}

type requestItem struct {
	ip     uint16
	domain string
}

// NewBogusDNS creates a new BogusDNS Server
func NewBogusDNS(addr string, prefix net.IP, ttl time.Duration) *BogusDNS {
	ret := &BogusDNS{
		srv:         dns.Server{Addr: addr, Net: "udp"},
		requestChan: make(chan *requestItem),
		IPPrefix:    prefix,
		DNSTTL:      ttl,
		TTL:         2 * ttl,
	}
	ret.srv.Handler = ret
	return ret
}

// ListenAndServe at the network
func (s *BogusDNS) ListenAndServe() error {
	go s.asgnRoutine()
	return s.srv.ListenAndServe()
}

// Shutdown the server gracefully
func (s *BogusDNS) Shutdown() error {
	close(s.requestChan)
	return s.Shutdown()
}

// Close the server forcefully
func (s *BogusDNS) Close() error {
	close(s.requestChan)
	return s.Close()
}

// This would be run in a single routine
// to prevent race condition.
func (s *BogusDNS) asgnRoutine() {
	for {
		req, ok := <-s.requestChan
		if !ok {
			break
		}
		// Assignement
		s.IPIndex[req.ip] = &bogusItem{
			domain: req.domain,
			ctime:  time.Now(),
		}
	}
}

// try to assign the item
func (s *BogusDNS) tryAssign(ip uint16, domain string) bool {
	current := s.IPIndex[ip]
	// If the IP is occupied
	if current != nil && current.domain != domain &&
		current.ctime.Add(s.TTL).Before(time.Now()) {
		return false
	}
	s.requestChan <- &requestItem{ip: ip, domain: domain}
	return true
}

func (s *BogusDNS) toIP(ip uint16) net.IP {
	result := make([]byte, len(s.IPPrefix))
	copy(result, s.IPPrefix)
	result[len(result)-2] = byte(ip / 256)
	result[len(result)-1] = byte(ip % 256)
	return result
}

func (s *BogusDNS) fromIP(ip net.IP) (uint16, error) {
	if ip == nil {
		return 0, fmt.Errorf("BogusDNS: IP cannot be nil")
	}
	if len(ip) != len(s.IPPrefix) {
		return 0, fmt.Errorf("BogusDNS: Not a valid IP address")
	}
	// Compare the prefix part
	for i := len(ip) - 2; i >= 0; i-- {
		if ip[i] != s.IPPrefix[i] {
			return 0, fmt.Errorf("BogusDNS: Not a valid IP address")
		}
	}
	return uint16(ip[len(ip)-2])*256 + uint16(ip[len(ip)-1]), nil
}

// GetIP address by the given address
func (s *BogusDNS) GetIP(domain string) (net.IP, error) {
	// Hashing the domain
	h := fnv.New64a()
	h.Write([]byte(domain))
	seed := h.Sum64()
	// Check the map
	for i := uint(0); i < 64; i++ {
		ip := uint16((seed>>i + seed<<(64-i)) & 0xFFFF) // take different digits from hash
		// Not found in the index
		if s.IPIndex[ip] == nil {
			log.Debugf("BogusDNS: try to assign %s to %d", domain, ip)
			if s.tryAssign(ip, domain) {
				return s.toIP(ip), nil
			}
		}
		if s.IPIndex[ip].domain == domain {
			// Refresh the item
			log.Debugf("BogusDNS: refresh the item %s to %d", domain, ip)
			if s.tryAssign(ip, domain) {
				return s.toIP(ip), nil
			}
		}
	}
	return nil, fmt.Errorf("BogusDNS: failed to assign an IP to the domain")
}

// GetAddress from the IP address given
// It can be void
func (s *BogusDNS) GetAddress(ip net.IP) (string, error) {
	// translate IP
	if ip == nil {
		return "", fmt.Errorf("BogusDNS: IP cannot be nil")
	}
	index, err := s.fromIP(ip)
	if err != nil {
		return "", err
	}
	i := s.IPIndex[index]
	if i == nil {
		return "", fmt.Errorf("BogusDNS: IP is not assigned to a corresponding address")
	}
	return i.domain, nil
}

// ServeDNS will handle the DNS request
func (s *BogusDNS) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	switch r.Question[0].Qtype {
	case dns.TypeA:
		msg.Authoritative = true
		domain := msg.Question[0].Name
		if domain[len(domain)-1] == '.' {
			domain = domain[:len(domain)-1]
		}
		ip, err := s.GetIP(domain)
		if err == nil {
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   msg.Question[0].Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    uint32(s.DNSTTL.Seconds()),
				},
				A: ip,
			})
		} else {
			log.Error(err)
		}
	}
	w.WriteMsg(&msg)
}
