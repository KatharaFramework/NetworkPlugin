module github.com/KatharaFramework/NetworkPlugin

replace github.com/KatharaFramework/NetworkPluginLib => ./lib/

go 1.21

require (
	github.com/KatharaFramework/NetworkPluginLib v0.0.0-00010101000000-000000000000
	github.com/docker/docker v26.0.0+incompatible
	github.com/docker/go-plugins-helpers v0.0.0-20211224144127-6eecb7beb651
)

require (
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/containernetworking/plugins v1.4.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/ishidawataru/sctp v0.0.0-20230406120618-7ff4192f6ff2 // indirect
	github.com/vishvananda/netlink v1.2.1-beta.2 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/tools v0.20.0 // indirect
)
