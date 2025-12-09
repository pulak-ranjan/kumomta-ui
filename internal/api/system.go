package api

import (
	"net"
	"net/http"
)

// GET /api/system/ips
// Returns a list of available IPv4 addresses on the server (excluding localhost)
func (s *Server) handleGetSystemIPs(w http.ResponseWriter, r *http.Request) {
	var ips []string

	ifaces, err := net.Interfaces()
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list interfaces"})
		return
	}

	for _, i := range ifaces {
		// Skip down interfaces or loopback
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := i.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Only support IPv4 for now, and skip loopback/link-local
			if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
				ips = append(ips, ip.String())
			}
		}
	}

	writeJSON(w, http.StatusOK, ips)
}
