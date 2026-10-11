package security

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const realClientIPHeader = "X-Real-Client-IP"

// UseRealClientIP makes gin.ClientIP return the visitor address behind Caddy
// or the in-container Cloudflare tunnel. Forwarded headers are honored only
// when the immediate peer is a loopback or private proxy.
func UseRealClientIP(r *gin.Engine) {
	_ = r.SetTrustedProxies([]string{
		"127.0.0.0/8",
		"::1/128",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fc00::/7",
	})
	r.TrustedPlatform = realClientIPHeader
	r.Use(func(c *gin.Context) {
		if ip := realClientIP(c.Request); ip != "" {
			c.Request.Header.Set(realClientIPHeader, ip)
		} else {
			c.Request.Header.Del(realClientIPHeader)
		}
		c.Next()
	})
}

func realClientIP(r *http.Request) string {
	peer := remoteHost(r.RemoteAddr)
	if !trustedProxy(peer) {
		return ""
	}
	if ip := publicIP(r.Header.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}
	if ip := publicIP(r.Header.Get("True-Client-IP")); ip != "" {
		return ip
	}
	if ip := validIP(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			if ip := validIP(parts[i]); ip != "" && !trustedProxy(ip) {
				return ip
			}
		}
	}
	return ""
}

func remoteHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return strings.Trim(remoteAddr, "[]")
	}
	return host
}

func validIP(value string) string {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return ""
	}
	return ip.String()
}

func publicIP(value string) string {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil || trustedProxy(ip.String()) || !ip.IsGlobalUnicast() {
		return ""
	}
	return ip.String()
}

func trustedProxy(value string) bool {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}
