LIBS=-lvdeplug -lpthread

.PHONY: extbuild

extbuild:
	apk add -U --no-cache gcc libc-dev clang llvm musl-dev linux-headers git automake autoconf libtool make cmake
	cd /tmp/ && git clone https://github.com/rd235/s2argv-execs.git
	cd /tmp/s2argv-execs/ && autoreconf -if && ./configure && make && make install
	cd /tmp/ && rm -rf s2argv-execs/ && git clone https://github.com/rd235/vdeplug4.git
	cd /tmp/vdeplug4 && sed -i 's/ | RTLD_DEEPBIND//' ./libvdeplug4/libvdeplug.c && mkdir build && cd build && cmake .. && make && make install && cd ../.. && rm -rf vdeplug4/
	
vde_ext.o: vde_ext.c
	$(CC) $(CFLAGS) -c $< -o $@

vde_ext: vde_ext.o
	$(CC) $(CFLAGS) -o $@ $^ $(LIBS)