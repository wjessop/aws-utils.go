package client

import (
	"context"
	"net"
	"net/http"
	"time"
)

var log logger

type logger interface {
	Fatalf(string, ...interface{})
	Infof(string, ...interface{})
	Debugf(string, ...interface{})
}

// New returns an HTTP client bound to the provided interface if provided
func New(iface string, theLogger logger) *http.Client {
	log = theLogger
	client := &http.Client{}

	if iface != "" {
		rt := &http.Transport{
			DialContext:           customDialer(iface),
			TLSHandshakeTimeout:   30 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			Proxy: http.ProxyFromEnvironment,
		}

		client.Transport = rt
	}

	return client
}

func customDialer(iface string) func(ctx context.Context, network string, addr string) (conn net.Conn, err error) {
	return func(ctx context.Context, network string, addr string) (conn net.Conn, err error) {
		ief, err := net.InterfaceByName(iface)
		if err != nil {
			log.Fatalf("Couldn't get interface by name, error: %s", err)
		}
		addrs, err := ief.Addrs()
		if err != nil {
			log.Fatalf("Couldn't get interface addresses, error: ", err)
		}

		for _, ip := range addrs {
			tcpAddr := &net.TCPAddr{
				IP: ip.(*net.IPNet).IP,
			}

			d := new(net.Dialer)
			d.LocalAddr = tcpAddr

			conn, err = d.DialContext(ctx, network, addr)
			if err == nil {
				log.Infof("Connected to AWS (%s) using local interface (%s) and address (%s)", addr, iface, tcpAddr)
				break
			}

			log.Debugf("Attempted to connect to AWS (%s) using interface (%s) local address (%s) as source but got error %s", addr, iface, tcpAddr, err)
		}
		return
	}
}
