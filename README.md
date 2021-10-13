# Katharà Network Plugin

This repository contains the Golang source code for the `kathara/katharanp` Docker Network Plugin.

This plugin creates pure L2 LANs using bridges and veths. 

This plugin is compiled and released on the DockerHub for both `amd64` and `arm64` architectures. The `tag` of the plugin is used to identify the architecture. For compatibility with previous versions, the `latest` tag is released as the `amd64` version.

To avoid assigning any IP subnet you **MUST** use `--ipam-driver=null` when creating networks with `kathara/katharanp` plugin. Otherwise, the veth endpoint inside the container will always receive an IP address from the default pool.

## Build from the source

To build the plugin, type on terminal:
```
$ make all_<arch>
```

where `<arch>` could be amd64 or arm64.

The build process uses Docker to build, so you don't need any dependencies installed in your machine.
