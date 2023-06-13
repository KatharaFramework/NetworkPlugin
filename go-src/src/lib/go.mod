module github.com/KatharaFramework/NetworkPluginLib

replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.8.1

go 1.20

require (
	github.com/google/uuid v1.3.0
	github.com/vishvananda/netlink v1.1.0
)

require (
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
)
