#!/usr/bin/make -f

PLUGIN_NAME?=kathara/katharanp
PLUGIN_CONTAINER=katharanp
PLUGIN_TMP_DIR=./tmp
PLUGIN_TMP_ROOTFS_DIR=./tmp/rootfs
ARCH?=amd64

.PHONY: all test clean gobuild image plugin

all: test clean gobuild plugin

test:
	cat config.json | python3 -m json.tool

clean:
	docker plugin rm -f ${PLUGIN_NAME}:${ARCH} || true
	docker rm -f ${PLUGIN_CONTAINER}_rootfs || true
	rm -rf ${PLUGIN_TMP_DIR}
	rm -rf katharanp

gobuild:%:
	go mod download
	GOOS=linux GOARCH=$* go build src/katharanp.go src/common_utils.go src/bridge_utils.go src/veth_utils.go

gobuild_docker_%:
	docker run -ti --rm -v `pwd`/go-src/:/root/go-src golang /bin/bash -c "cd /root/go-src && make gobuild_$*"

image: 
	mkdir -p ${PLUGIN_TMP_ROOTFS_DIR}
	cp ./Dockerfile ${PLUGIN_TMP_DIR}/Dockerfile
	cp ./entrypoint.sh ${PLUGIN_TMP_DIR}/entrypoint.sh
	cp ./katharanp ${PLUGIN_TMP_DIR}/
	docker buildx build --platform linux/${ARCH} --load -t ${PLUGIN_CONTAINER}:rootfs ${PLUGIN_TMP_DIR}
	docker create --name ${PLUGIN_CONTAINER}_rootfs ${PLUGIN_CONTAINER}:rootfs
	docker export ${PLUGIN_CONTAINER}_rootfs | tar -x -C ${PLUGIN_TMP_ROOTFS_DIR}
	cp config.json ${PLUGIN_TMP_DIR}/config.json
	docker rm -vf ${PLUGIN_CONTAINER}_rootfs
	docker rmi ${PLUGIN_CONTAINER}:rootfs
	rm ${PLUGIN_TMP_DIR}/Dockerfile ${PLUGIN_TMP_DIR}/entrypoint.sh ${PLUGIN_TMP_DIR}/katharanp

plugin: image
	docker plugin create ${PLUGIN_NAME}:${ARCH} ${PLUGIN_TMP_DIR}
