# Katharà Network Plugin

This repository contains the Golang source code for the `kathara/katharanp` Docker Network Plugin.

This plugin creates pure L2 LANs using bridges and veths. 

To avoid assigning any IP subnet you **MUST** use `--ipam-driver=null` when creating networks with `kathara/katharanp` plugin. Otherwise, the veth endpoint inside the container will always receive an IP address from the default pool.

## Build from the source

The plugin is compiled and released for both `amd64` and `arm64` architectures. The tag of the plugin identifies the architecture. For backward compatibility, the `latest` tag is a retag of the `amd64` version.

To build the plugin, type on terminal:
```
$ make all_<arch>
```

Where `<arch>` can be: `amd64` or `arm64`.

The build process leverages on Docker, so you don't need any dependencies installed in your machine.

## Use `katharanp` without Katharà

It is possible to leverage on `katharanp` as a standalone Docker Network Plugin, in order to create pure L2 networks.

Note that using nftables as the iptables backend requires a `xtables.lock` file in order to work properly. Hence the same host lock should be shared (and hence, mounted) with the network plugin container. 

So, install the plugin passing the path of your `xtables.lock` file to the plugin configuration, with the following command:
```bash
docker plugin install kathara/katharanp:amd64 xtables_lock.source="/var/run/xtables.lock"
# or
docker plugin install kathara/katharanp:arm64 xtables_lock.source="/var/run/xtables.lock"
```

At this point, you can create a network using the standard Docker command:
```bash
docker network create --driver=kathara/katharanp:amd64 --ipam-driver=null l2net
# or
docker network create --driver=kathara/katharanp:arm64 --ipam-driver=null l2net
```
