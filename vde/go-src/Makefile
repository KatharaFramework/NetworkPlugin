#!/usr/bin/make -f

.PHONY: gobuild_%

gobuild_%:
	apk add -U --no-cache gcc libc-dev clang llvm musl-dev linux-headers git automake autoconf libtool make cmake
	cd /tmp/ && git clone https://github.com/rd235/s2argv-execs.git
	cd /tmp/s2argv-execs/ && autoreconf -if && ./configure && make && make install
	cd /tmp/ && rm -rf s2argv-execs/ && git clone https://github.com/rd235/vdeplug4.git && cd vdeplug4 && git checkout 3a12bea918e88e86af8fe75501d33374de3115ec
	cd /tmp/vdeplug4 && sed -i 's/ | RTLD_DEEPBIND//' ./libvdeplug4/libvdeplug.c && sed -i '1i#include <libgen.h>' srcvdeplug4/vde_plug.c && mkdir build && cd build && cmake .. && make && make install && cd ../.. && rm -rf vdeplug4/
	cd src/lib && go mod download && go test . && GOOS=linux GOARCH=$* go build .
	cd src && go mod download && GOOS=linux GOARCH=$* go build -v katharanp.go  
	chmod 777 ./src/katharanp
	