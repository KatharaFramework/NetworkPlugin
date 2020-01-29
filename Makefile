#!/usr/bin/make -f

PLUGIN_NAME=kathara/katharanp
PLUGIN_CONTAINTER=katharanp
PLUGIN_TMP_DIR=./tmp
PLUGIN_TMP_ROOTFS_DIR=./tmp/rootfs

.PHONY: all_stretch all_buster test_% clean_% gobuild image_% plugin_%

all_stretch: test_stretch clean_stretch gobuild plugin_stretch
all_buster: test_buster clean_buster gobuild plugin_buster

test_%:
	cat config_$*.json | python3 -m json.tool

clean_%:
	docker plugin rm -f ${PLUGIN_NAME}:$* || true
	docker rm -f ${PLUGIN_CONTAINTER}_rootfs_$* || true
	rm -rf ${PLUGIN_TMP_DIR}
	rm -rf katharanp

gobuild:
	go get -v github.com/docker/libnetwork github.com/vishvananda/netlink github.com/docker/go-plugins-helpers/network github.com/google/uuid
	go build src/katharanp.go src/common_utils.go src/iptables_utils.go src/bridge_utils.go src/veth_utils.go

image_%: 
	mkdir -p ${PLUGIN_TMP_ROOTFS_DIR}
	cp ./Dockerfile ${PLUGIN_TMP_DIR}/Dockerfile
	cp ./entrypoint.sh ${PLUGIN_TMP_DIR}/entrypoint.sh
	cp ./katharanp ${PLUGIN_TMP_DIR}/katharanp
	sed -i -e "s|__RELEASE__|$*|g" ${PLUGIN_TMP_DIR}/Dockerfile
	docker build -t ${PLUGIN_CONTAINTER}:rootfs_$* ${PLUGIN_TMP_DIR}
	docker create --name ${PLUGIN_CONTAINTER}_rootfs_$* ${PLUGIN_CONTAINTER}:rootfs_$*
	docker export ${PLUGIN_CONTAINTER}_rootfs_$* | tar -x -C ${PLUGIN_TMP_ROOTFS_DIR}
	cp config_$*.json ${PLUGIN_TMP_DIR}/config.json
	docker rm -vf ${PLUGIN_CONTAINTER}_rootfs_$*
	docker rmi ${PLUGIN_CONTAINTER}:rootfs_$*
	rm ${PLUGIN_TMP_DIR}/Dockerfile ${PLUGIN_TMP_DIR}/entrypoint.sh ${PLUGIN_TMP_DIR}/katharanp

plugin_%: image_%
	docker plugin create ${PLUGIN_NAME}:$* ${PLUGIN_TMP_DIR}