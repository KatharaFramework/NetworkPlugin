module github.com/KatharaFramework/NetworkPluginLib

replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.8.1

go 1.20

require (
	github.com/containernetworking/plugins v1.3.0
	github.com/google/uuid v1.3.0
	github.com/vishvananda/netlink v1.2.1-beta.2
)

require (
	github.com/vishvananda/netns v0.0.4 // indirect
	golang.org/x/sys v0.7.0 // indirect
)
