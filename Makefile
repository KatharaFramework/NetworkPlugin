#!/usr/bin/make -f

PLUGIN_NAME=kathara/katharanp
PLUGIN_CONTAINTER=katharanp
PLUGIN_TMP_DIR=./tmp
PLUGIN_TMP_ROOTFS_DIR=./tmp/rootfs

.PHONY: all test clean gobuild image plugin

all: test clean gobuild plugin

test:
	cat config.json | python3 -m json.tool

clean:
	docker plugin rm -f ${PLUGIN_NAME} || true
	docker rm -f ${PLUGIN_CONTAINTER}_rootfs || true
	rm -rf ${PLUGIN_TMP_DIR}
	rm -rf katharanp

gobuild:
	go get -v github.com/docker/libnetwork github.com/vishvananda/netlink github.com/docker/go-plugins-helpers/network github.com/google/uuid
	go build src/katharanp.go src/common_utils.go src/bridge_utils.go src/veth_utils.go

image: 
	mkdir -p ${PLUGIN_TMP_ROOTFS_DIR}
	cp ./Dockerfile ${PLUGIN_TMP_DIR}/Dockerfile
	cp ./entrypoint.sh ${PLUGIN_TMP_DIR}/entrypoint.sh
	cp ./katharanp ${PLUGIN_TMP_DIR}/katharanp
	docker build -t ${PLUGIN_CONTAINTER}:rootfs ${PLUGIN_TMP_DIR}
	docker create --name ${PLUGIN_CONTAINTER}_rootfs ${PLUGIN_CONTAINTER}:rootfs
	docker export ${PLUGIN_CONTAINTER}_rootfs | tar -x -C ${PLUGIN_TMP_ROOTFS_DIR}
	cp config.json ${PLUGIN_TMP_DIR}/config.json
	docker rm -vf ${PLUGIN_CONTAINTER}_rootfs
	docker rmi ${PLUGIN_CONTAINTER}:rootfs
	rm ${PLUGIN_TMP_DIR}/Dockerfile ${PLUGIN_TMP_DIR}/entrypoint.sh ${PLUGIN_TMP_DIR}/katharanp

plugin: image
	docker plugin create ${PLUGIN_NAME} ${PLUGIN_TMP_DIR}
