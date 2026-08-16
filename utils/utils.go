package utils

import (
	"net/http"
	"strings"
)

func GetClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		ips := strings.Split(ip, ",")
		return strings.TrimSpace(ips[0])
	}

	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	remoteIP := r.RemoteAddr
	if idx := strings.LastIndex(remoteIP, ":"); idx != -1 {
		remoteIP = remoteIP[:idx]
	}

	remoteIP = strings.Trim(remoteIP, "[]")

	if remoteIP == "::1" {
		return "127.0.0.1"
	}

	return remoteIP
}

func ParseTokenName(userAgent string) string {
	if userAgent == "" {
		return "API Session"
	}

	ua := strings.ToLower(userAgent)
	switch {
	case strings.Contains(ua, "postman"):
		return "Postman Client"
	case strings.Contains(ua, "curl"):
		return "cURL CLI"
	case strings.Contains(ua, "mobile"), strings.Contains(ua, "android"), strings.Contains(ua, "iphone"):
		return "Mobile Session"
	case strings.Contains(ua, "chrome"):
		return "Chrome Browser"
	case strings.Contains(ua, "firefox"):
		return "Firefox Browser"
	case strings.Contains(ua, "safari"):
		return "Safari Browser"
	default:
		return "Web Session"
	}
}