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
		return plugin.Error("policy", err)
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
	args := c.RemainingArgs()
	// <label> <definition>
	// <label> edns0 <id>
	// <label> ends0 <id> <encoded-format> <params of format ...>
	// Valid encoded-format are hex (default), bytes, ip.

	argsLen := len(args)
	if argsLen != 3 && argsLen != 4 && argsLen != 7 {
		return fmt.Errorf("Invalid edns0 directive")
	}
	name := args[0]
	code := args[2]

	dataType := "hex"
	size := "0"
	start := "0"
	end := "0"

	if argsLen > 3 {
		dataType = args[3]
	}

	if argsLen == 7 && dataType == "hex" {
		size = args[4]
		start = args[5]
		end = args[6]
	}

	err := p.addEDNS0Map(code, name, dataType, size, start, end)
	if err != nil {
		return fmt.Errorf("Could not add EDNS0 map for %s: %s", args[0], err)
	}

	return nil
}

func (p *requestPlugin) addEDNS0Map(code, name, dataType, sizeStr, startStr, endStr string) error {
	c, err := strconv.ParseUint(code, 0, 16)
	if err != nil {
		return fmt.Errorf("Could not parse EDNS0 code: %s", err)
	}
	size, err := strconv.ParseUint(sizeStr, 10, 32)
	if err != nil {
		return fmt.Errorf("Could not parse EDNS0 data size: %s", err)
	}
	start, err := strconv.ParseUint(startStr, 10, 32)
	if err != nil {
		return fmt.Errorf("Could not parse EDNS0 start index: %s", err)
	}
	end, err := strconv.ParseUint(endStr, 10, 32)
	if err != nil {
		return fmt.Errorf("Could not parse EDNS0 end index: %s", err)
	}
	if end <= start && end != 0 {
		return fmt.Errorf("End index should be > start index (actual %d <= %d)", end, start)
	}
	if end > size && size != 0 {
		return fmt.Errorf("End index should be <= size (actual %d > %d)", end, size)
	}
	ednsType, ok := stringToEDNS0MapType[dataType]
	if !ok {
		return fmt.Errorf("Invalid dataType for EDNS0 map: %s", dataType)
	}
	ecode := uint16(c)
	p.options[ecode] = append(p.options[ecode], &edns0Map{name, ecode, ednsType, uint(size), uint(start), uint(end)})
	return nil
}