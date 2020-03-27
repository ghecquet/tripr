package resolver

import (
	"bytes"
	"log"
	"net"
	"strings"

	"google.golang.org/grpc/resolver"
)

var (
	watchers  []func(string, []string)
	endpoints = make(map[string][]string)
)

func init() {
	resolver.Register(&cellsBuilder{})
	watch()
}

func watch() {
	addr, err := net.ResolveUDPAddr("udp", discoveryAddr)
	if err != nil {
		log.Fatal(err)
	}
	l, err := net.ListenMulticastUDP("udp", nil, addr)
	l.SetReadBuffer(maxDatagramSize)

	go func() {
		for {
			b := make([]byte, maxDatagramSize)
			n, src, err := l.ReadFromUDP(b)
			if err != nil {
				log.Fatal("ReadFromUDP failed:", err)
			}

			if n == 0 {
				continue
			}

			v := strings.Split(string(bytes.Trim(b, "\x00")), ",")
			services := v[1:]

			host, _, _ := net.SplitHostPort(src.String())
			_, port, _ := net.SplitHostPort(v[0])

			addr := net.JoinHostPort(host, port)

			for _, service := range services {
				current, ok := endpoints[service]
				if !ok {
					endpoints[service] = []string{addr}
					continue
				}

				found := false
				for _, c := range current {
					if c == addr {
						found = true
						break
					}
				}

				if !found {
					endpoints[service] = append(endpoints[service], addr)
				}

				for _, watcher := range watchers {
					watcher(service, endpoints[service])
				}
			}
		}
	}()
}

const (
	scheme          = "cells"
	discoveryAddr   = "224.0.0.1:9999"
	maxDatagramSize = 8192
)

type cellsBuilder struct{}

func (*cellsBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &cellsResolver{
		target: target,
		cc:     cc,
		rn:     make(chan struct{}, 1),
	}
	go r.watch()
	r.ResolveNow(resolver.ResolveNowOptions{})
	return r, nil
}

func (*cellsBuilder) Scheme() string {
	return scheme
}

type cellsResolver struct {
	target resolver.Target
	cc     resolver.ClientConn
	rn     chan struct{}
}

func (r *cellsResolver) watch() {
	watchers = append(watchers, func(s string, eps []string) {
		if s == r.target.Endpoint {
			addresses := []resolver.Address{}
			for _, ep := range eps {
				addresses = append(addresses, resolver.Address{Addr: ep})
			}

			r.cc.UpdateState(resolver.State{Addresses: addresses})
			r.rn <- struct{}{}
		}
	})
}

func (r *cellsResolver) ResolveNow(o resolver.ResolveNowOptions) {
	<-r.rn
}

func (*cellsResolver) Close() {}
