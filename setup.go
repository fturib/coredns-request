package request

import (
	"fmt"
	"strconv"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"

	"github.com/mholt/caddy"
)

func init() {
	caddy.RegisterPlugin("request", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	r, err := parseRequest(c)

	if err != nil {
		return plugin.Error("request", err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		r.Next = next
		return r
	})

	return nil
}

func parseRequest(c *caddy.Controller) (*requestPlugin, error) {
	r := newRequestPlugin()
	for c.Next() {
		c.RemainingArgs()
		for c.NextBlock() {
			err := r.parseEDNS0(c)
			if err != nil {
				return nil, err
			}
		}
	}
	return r, nil
}

func (p *requestPlugin) parseEDNS0(c *caddy.Controller) error {
	name := c.Val()
	args := c.RemainingArgs()
	// <label> <definition>
	// <label> edns0 <id>
	// <label> ends0 <id> <encoded-format> <params of format ...>
	// Valid encoded-format are hex (default), bytes, ip.

	argsLen := len(args)
	if argsLen != 2 && argsLen != 3 && argsLen != 6 {
		return fmt.Errorf("Invalid edns0 directive")
	}
	code := args[1]

	dataType := "hex"
	size := "0"
	start := "0"
	end := "0"

	if argsLen > 2 {
		dataType = args[2]
	}

	if argsLen == 6 && dataType == "hex" {
		size = args[3]
		start = args[4]
		end = args[5]
	}

	err := p.addEDNS0Map(code, name, dataType, size, start, end)
	if err != nil {
		return fmt.Errorf("Could not add EDNS0 map for %s: %s", name, err)
	}

	return nil
}

func newEDNS0Map(code, name, dataType, sizeStr, startStr, endStr string) (*edns0Map, error) {
	c, err := strconv.ParseUint(code, 0, 16)
	if err != nil {
		return nil, fmt.Errorf("Could not parse EDNS0 code: %s", err)
	}
	size, err := strconv.ParseUint(sizeStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("Could not parse EDNS0 data size: %s", err)
	}
	start, err := strconv.ParseUint(startStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("Could not parse EDNS0 start index: %s", err)
	}
	end, err := strconv.ParseUint(endStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("Could not parse EDNS0 end index: %s", err)
	}
	if end <= start && end != 0 {
		return nil, fmt.Errorf("End index should be > start index (actual %d <= %d)", end, start)
	}
	if end > size && size != 0 {
		return nil, fmt.Errorf("End index should be <= size (actual %d > %d)", end, size)
	}
	ednsType, ok := stringToEDNS0MapType[dataType]
	if !ok {
		return nil, fmt.Errorf("Invalid dataType for EDNS0 map: %s", dataType)
	}
	ecode := uint16(c)
	return &edns0Map{name, ecode, ednsType, uint(size), uint(start), uint(end)}, nil
}

func (p *requestPlugin) addEDNS0Map(code, name, dataType, sizeStr, startStr, endStr string) error {
	m, err := newEDNS0Map(code, name, dataType, sizeStr, startStr, endStr)
	if err != nil {
		return err
	}
	p.options[m.code] = append(p.options[m.code], m)
	return nil
}
