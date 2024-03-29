module github.com/KatharaFramework/NetworkPlugin

replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.8.1

replace github.com/KatharaFramework/NetworkPluginLib => ./lib/

go 1.20

require (
	github.com/KatharaFramework/NetworkPluginLib v0.0.0-00010101000000-000000000000
	github.com/docker/go-plugins-helpers v0.0.0-20210623094020-7ef169fb8b8e
	github.com/docker/libnetwork v0.8.0-dev.2.0.20210525090646-64b7a4574d14
)

require (
	github.com/Microsoft/go-winio v0.6.0 // indirect
	github.com/containernetworking/plugins v1.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/ishidawataru/sctp v0.0.0-20210707070123-9a39160e9062 // indirect
	github.com/vishvananda/netlink v1.2.1-beta.2 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	golang.org/x/mod v0.9.0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/tools v0.7.0 // indirect
	gotest.tools/v3 v3.0.3 // indirect
)
