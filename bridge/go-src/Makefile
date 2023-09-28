#!/usr/bin/make -f

.PHONY: gobuild_%

gobuild_%:
	go mod download
	GOOS=linux GOARCH=$* go build src/katharanp.go src/common_utils.go src/bridge_utils.go src/veth_utils.go
	chmod 777 ./katharanp
	