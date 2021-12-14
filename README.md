# Katharà Network Plugin

This repository contains the Golang source code for the `kathara/katharanp` Docker Network Plugin.

This plugin creates pure L2 LANs using bridges and veths. 

To avoid assigning any IP subnet you **MUST** use `--ipam-driver=null` when creating networks with `kathara/katharanp` plugin. Otherwise, the veth endpoint inside the container will always receive an IP address from the default pool.

## Build from the source

The plugin is compiled and released for both `amd64` and `arm64` architectures. The tag of the plugin identifies the architecture. For compatibility with previous Katharà versions, the `latest` tag is a retag of the `amd64` version.

To build the plugin, type on terminal:
```
$ make all_<arch>
```

Where `<arch>` can be: `amd64` or `arm64`.

The build process leverages on Docker, so you don't need any dependencies installed in your machine.
