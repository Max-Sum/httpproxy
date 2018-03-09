package client

import (
	"bufio"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// GFWList is a extracted gfwlist
type GFWList struct {
	Blacklist []string
	Whitelist []string
}

// NewGFWList returns an empty GFWList with default size
func NewGFWList() *GFWList {
	return &GFWList{
		Blacklist: make([]string, 0, 6000),
		Whitelist: make([]string, 0, 500),
	}
}

// Update the GFWList and extract it.
func (l *GFWList) Update(url string, tr http.RoundTripper) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := tr.RoundTrip(req)
	if err != nil {
		return err
	}
	// Extract from the response
	defer resp.Body.Close()
	return l.Extract(resp.Body, true)
}

// Extract rules from the input reader
func (l *GFWList) Extract(r io.Reader, clear bool) error {
	if clear && len(l.Blacklist)+len(l.Whitelist) > 0 {
		log.Debugf("GFWList: clear the list, with %d and %d items in the lists", len(l.Blacklist), len(l.Whitelist))
		l = NewGFWList()
	}
	decoder := base64.NewDecoder(base64.StdEncoding, r)
	scanner := bufio.NewScanner(decoder)
	for scanner.Scan() {
		s := scanner.Text()
		// Strip-off comments
		s = strings.SplitN(s, "!", 2)[0]
		// Empty line
		if s == "" || strings.HasPrefix(s, "[") {
			continue
		}
		if strings.HasPrefix(s, "@@") {
			l.Whitelist = append(l.Whitelist, s[2:])
		} else {
			l.Blacklist = append(l.Blacklist, s)
		}
	}
	return nil
}

// Match a url with the gfwlist
func (l *GFWList) Match(u *url.URL) bool {
	ou := *u
	// Strip away user and port
	ou.User = nil
	ou.Host = u.Hostname()
	// Not matching whitelist
	for _, rule := range l.Whitelist {
		if l.matchRule(&ou, rule) {
			log.Debugf("GFWList: Matched %s with rule %s in whitelist", ou.String(), rule)
			return false
		}
	}
	// matching blacklist
	for _, rule := range l.Blacklist {
		if l.matchRule(&ou, rule) {
			log.Debugf("GFWList: Matched %s with rule %s in blacklist", ou.String(), rule)
			return true
		}
	}
	return false
}

// MatchAddr use hostname and port to determine the result
// with best effort. (GFWList is not designed to do this)
// The port can be ignored.
func (l *GFWList) MatchAddr(host, port string) bool {
	// Not matching whitelist
	for _, rule := range l.Whitelist {
		if l.matchRuleAddr(host, port, rule, false) {
			log.Debugf("GFWList: Matched %s:%s with rule %s in whitelist", host, port, rule)
			return false
		}
	}
	// matching blacklist
	// blacklist will be fuzzy to match
	for _, rule := range l.Blacklist {
		if l.matchRuleAddr(host, port, rule, true) {
			log.Debugf("GFWList: Matched %s:%s with rule %s in blacklist", host, port, rule)
			return true
		}
	}
	return false
}

func (l *GFWList) matchRule(u *url.URL, rule string) bool {
	// domain suffix
	if strings.HasPrefix(rule, "||") {
		rule = rule[2:]
		return l.glob(u.Host, rule) || l.glob(u.Host, "*."+rule)
	}
	// URL prefix
	if strings.HasPrefix(rule, "|") {
		return l.glob(u.String(), rule[1:]+"*")
	}
	// regexp
	if strings.HasPrefix(rule, "/") && strings.HasSuffix(rule, "/") {
		r, err := regexp.Compile(rule[1 : len(rule)-1])
		if err != nil {
			return false
		}
		return r.MatchString(u.String())
	}
	// keyword
	return l.glob(u.String(), "*"+rule+"*")
}

func (l *GFWList) matchRuleAddr(host, port, rule string, fuzzy bool) bool {
	// domain suffix
	if strings.HasPrefix(rule, "||") {
		rule = rule[2:]
		return l.glob(host, rule) || l.glob(host, "*."+rule)
	}
	// Guessing the URL
	var url string
	if port == "443" || port == "https" {
		url = "https://" + host + "/"
	} else {
		url = "http://" + host + "/"
	}
	// URL prefix
	if strings.HasPrefix(rule, "|") {
		if fuzzy {
			// Trim all the requestURI part
			idx := strings.Index(rule, "://")
			if idx < 0 {
				idx = -3
			}
			search := strings.Index(rule[idx+3:], "/")
			if search >= 0 {
				rule = rule[:idx+3+search]
			}
		}
		return l.glob(url, rule[1:]+"*")
	}
	// regexp
	if strings.HasPrefix(rule, "/") && strings.HasSuffix(rule, "/") {
		r, err := regexp.Compile(rule[1 : len(rule)-1])
		if err != nil {
			return false
		}
		return r.MatchString(url)
	}
	// keyword
	if fuzzy {
		// Trim all the requestURI part
		idx := strings.Index(rule, "://")
		if idx < 0 {
			idx = -3
		}
		search := strings.Index(rule[idx+3:], "/")
		if search >= 0 {
			rule = rule[:idx+3+search]
		}
	}
	return l.glob(url, "*"+rule+"*")
}

func (l *GFWList) glob(s, pattern string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) < 2 {
		return s == pattern
	}
	// Match starting part
	if !strings.HasPrefix(s, parts[0]) {
		return false
	}
	s = s[len(parts[0]):]
	for i := 1; i < len(parts)-1; i++ {
		idx := strings.Index(s, parts[i])
		// Check that the middle parts match.
		if idx < 0 {
			return false
		}
		// Trim the matched part
		s = s[idx+len(parts[i]):]
	}
	return strings.HasSuffix(s, parts[len(parts)-1])
}

// ExportDomains exports two black and white list of domains
// with best effort (GFWList is not designed to do this)
func (l *GFWList) ExportDomains() ([]string, []string) {
	// Domain regexp
	domainRegexp := regexp.MustCompile("^(\\.?([^\\/:@\\*\\.]+\\.)+[^\\/:@\\*\\.]+)/?$")
	fuzzyDomainRegexp := regexp.MustCompile("([^\\/:@\\*\\.]+\\.)+[^\\/:@\\*\\.]+")
	whitelist := make([]string, 0, len(l.Whitelist))
	blacklist := make([]string, 0, len(l.Blacklist))
	// Deduplicate some domains
	tmp := ""
	for _, rule := range l.Whitelist {
		if strings.HasPrefix(rule, "||") {
			rule = rule[2:]
			// No way to use a rule with an asterisk
			if !strings.ContainsRune(rule, '*') && tmp != rule {
				whitelist = append(whitelist, rule)
				tmp = rule
			}
			continue
		}
		// regexp
		if strings.HasPrefix(rule, "/") && strings.HasSuffix(rule, "/") {
			// No way to use it now
			continue
		}
		// URL Prefix
		if strings.HasPrefix(rule, "|") {
			rule = rule[1:]
		}
		// keyword
		matches := domainRegexp.FindStringSubmatch(rule)
		if len(matches) > 1 && tmp != matches[1] {
			whitelist = append(whitelist, matches[1])
			tmp = matches[1]
		}
		continue
	}
	tmp = ""
	for _, rule := range l.Blacklist {
		if strings.HasPrefix(rule, "||") {
			rule = rule[2:]
			// No way to use a rule with an asterisk
			if !strings.ContainsRune(rule, '*') && tmp != rule {
				blacklist = append(blacklist, rule)
				tmp = rule
			}
			continue
		}
		// regexp
		if strings.HasPrefix(rule, "/") && strings.HasSuffix(rule, "/") {
			// No way to use it now
			continue
		}
		// URL Prefix
		if strings.HasPrefix(rule, "|") {
			rule = rule[1:]
		}
		// keyword
		rule = fuzzyDomainRegexp.FindString(rule)
		if rule != "" && rule != tmp {
			blacklist = append(blacklist, rule)
			tmp = rule
		}
	}
	return blacklist, whitelist
}
