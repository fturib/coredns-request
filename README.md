# coredns-request
Plugin request for coding DNS msg information into metadata



### Request plugin

Should be regular plugin for CoreDNS
Could be the metadata plugin.
Nothing specific.
Will need to reuse the EDNS0 syntax for extraction

~~~
request {
      client_id edns0 0xffed
      group_id edns0 0xffee hex 16 0 16
      <label> edns0 <id>
      <label> ends0 <id> <encoded-format> <params of format ...>
}
~~~

so far, only 'hex' format is supported with params <length>  <start> <end>


currently supported metadata is based on the variables used in REWRITE section. these are:

	queryName  = "qname"
	queryType  = "qtype"
	clientIP   = "client_ip"
	clientPort = "client_port"
	protocol   = "protocol"
	serverIP   = "server_ip"
	serverPort = "server_port"

