package main

import (
	"net/http"
	"testing"
	"time"
)

func TestServerTimeoutsLeaveRoomForProtocolErrors(t *testing.T) {
	const requestTimeout = 20 * time.Second
	server := httpServer(":0", http.NewServeMux(), requestTimeout)
	if server.ReadHeaderTimeout != 2*time.Second {
		t.Fatalf("header timeout = %s", server.ReadHeaderTimeout)
	}
	if server.ReadTimeout < requestTimeout || server.WriteTimeout < requestTimeout+5*time.Second {
		t.Fatalf("read = %s write = %s", server.ReadTimeout, server.WriteTimeout)
	}
}
