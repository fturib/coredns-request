package request

import (
	"encoding/hex"

	"github.com/miekg/dns"

	"context"

	"net"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metadata"
	"github.com/coredns/coredns/request"
)

const (
	typeEDNS0Bytes = iota
	typeEDNS0Hex
	typeEDNS0IP
)

const (
	QueryName  = "qname"
	QueryType  = "qtype"
	ClientIP   = "client_ip"
	ClientPort = "client_port"
	Protocol   = "protocol"
	ServerIP   = "server_ip"
	ServerPort = "server_port"
	ResponseIP = "response_ip"
)

var stringToEDNS0MapType = map[string]uint16{
	"bytes":   typeEDNS0Bytes,
	"hex":     typeEDNS0Hex,
	"address": typeEDNS0IP,
}

type edns0Map struct {
	name     string
	code     uint16
	dataType uint16
	size     uint
	start    uint
	end      uint
}

// requestPlugin represents a plugin instance that can validate DNS
// requests and replies using PDP server.
type requestPlugin struct {
	Next    plugin.Handler
	options map[uint16][]*edns0Map
}

func newRequestPlugin() *requestPlugin {
	pol := &requestPlugin{options: make(map[uint16][]*edns0Map, 0)}
	return pol
}

// ServeDNS implements the Handler interface.
func (rq requestPlugin) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	return plugin.NextOrFailure(rq.Name(), rq.Next, ctx, w, r)

}

// Name implements the Handler interface.
func (m requestPlugin) Name() string { return "request" }

func (p *requestPlugin) Metadata(ctx context.Context, state request.Request) context.Context {
	return p.fillMetadata(ctx, state)
}

func (p *requestPlugin) fillMetadata(ctx context.Context, state request.Request) context.Context {

	p.declareMetadata(QueryName, state.QName(), ctx)
	p.declareMetadata(QueryType, dns.Type(state.QType()).String(), ctx)
	p.declareMetadata(ClientIP, state.IP(), ctx)
	//TOD - continue to fill the global variables.

	p.getAttrsFromEDNS0(state.Req, ctx)

	metadata.SetValueFunc(ctx, "request/"+ResponseIP, func() string {
		ip := getRespIP(state.Req)
		if ip != nil {
			return ip.String()
		}
		return ""
	})
	return ctx

}

func (p *requestPlugin) declareMetadata(name string, value string, ctx context.Context) {
	metadata.SetValueFunc(ctx, "request/"+name, func() string { return value })
}

func (p *requestPlugin) getAttrsFromEDNS0(r *dns.Msg, ctx context.Context) {
	o := r.IsEdns0()
	if o == nil {
		return
	}

	for _, opt := range o.Option {
		optLocal, local := opt.(*dns.EDNS0_LOCAL)
		if !local {
			continue
		}
		opts, ok := p.options[optLocal.Code]
		if !ok {
			continue
		}
		p.parseOptionGroup(optLocal.Data, opts, ctx)
	}
}

func (p *requestPlugin) parseOptionGroup(data []byte, options []*edns0Map, ctx context.Context) {
	for _, option := range options {
		var value string
		switch option.dataType {
		case typeEDNS0Bytes:
			value = string(data)
		case typeEDNS0Hex:
			value = parseHex(data, option)
		case typeEDNS0IP:
			ip := net.IP(data)
			value = ip.String()
		}
		if value != "" {
			p.declareMetadata(option.name, value, ctx)
		}
	}
}

func parseHex(data []byte, option *edns0Map) string {
	size := uint(len(data))
	// if option.size == 0 - don't check size
	if option.size > 0 {
		if size != option.size {
			// skip parsing option with wrong size
			return ""
		}
	}
	start := uint(0)
	if option.start < size {
		// set start index
		start = option.start
	} else {
		// skip parsing option if start >= data size
		return ""
	}
	end := size
	// if option.end == 0 - return data[start:]
	if option.end > 0 {
		if option.end <= size {
			// set end index
			end = option.end
		} else {
			// skip parsing option if end > data size
			return ""
		}
	}
	return hex.EncodeToString(data[start:end])
}

func getRespIP(r *dns.Msg) net.IP {
	if r == nil {
		return nil
	}

	var ip net.IP
	for _, rr := range r.Answer {
		switch rr := rr.(type) {
		case *dns.A:
			ip = rr.A

		case *dns.AAAA:
			ip = rr.AAAA
		}
	}

	return ip
}
