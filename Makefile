#!/usr/bin/make -f

.PHONY: create-builder delete-builder test all_arm64 all_push_arm64 all_amd64 all_push_amd64

clean_%: delete-builder
	cd bridge && make clean_$*
	cd vde && make clean_$*

all_arm64: create-builder
	cd bridge && make all_arm64
	cd vde && make all_arm64

all_push_arm64: create-builder
	cd bridge && make all_push_arm64
	cd vde && make all_push_arm64

all_amd64: create-builder
	cd bridge && make all_amd64
	cd vde && make all_amd64

all_push_amd64: create-builder
	cd bridge && make all_push_amd64
	cd vde && make all_push_amd64

create-builder: delete-builder
	docker buildx create --name kat-np-builder --use
	docker buildx inspect --bootstrap

delete-builder:
	docker buildx rm kat-np-builder || true