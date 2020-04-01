package resolver

import (
	"bytes"
	"log"
	"net"
	"strings"

	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/resolver"
)

var (
	watchers  []func(string, []string)
	dns       = make(map[string][]string)
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

			host, _, _ := net.SplitHostPort(src.String())

			req := &Request{}
			proto.Unmarshal(bytes.Trim(b, "\x00"), req)

			switch v := req.Request.(type) {
			case *Request_Service:
				_, port, _ := net.SplitHostPort(v.Service.Addr)
				addr := net.JoinHostPort(host, port)
				service := v.Service.Name

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
			case *Request_Dns:
				name := v.Dns.Name

				current, ok := dns[name]
				if !ok {
					dns[name] = []string{host}
					continue
				}

				found := false
				for _, c := range current {
					if c == host {
						found = true
						break
					}
				}

				if !found {
					dns[name] = append(dns[name], host)
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
		desc := strings.Split(r.target.Endpoint, "@")
		all := (len(desc) == 1)

		if (all && s == desc[0]) || s == desc[1] {
			addresses := []resolver.Address{}
			for _, ep := range eps {
				if all {
					addresses = append(addresses, resolver.Address{Addr: ep})
				} else {
					host, _, _ := net.SplitHostPort(ep)
					for _, ip := range dns[desc[0]] {
						if ip == host {
							addresses = append(addresses, resolver.Address{Addr: ep})
						}
					}
				}
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

func NewService(addr string, service string) *Request {
	return &Request{
		Request: &Request_Service{
			Service: &Service{
				Addr: addr,
				Name: service,
			},
		},
	}
}

func NewDNS(name string) *Request {
	return &Request{
		Request: &Request_Dns{
			Dns: &DNS{
				Name: name,
			},
		},
	}
}
