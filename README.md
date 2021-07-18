# Katharà Network Plugin

This repository contains the Golang source code for the `kathara/katharanp` Docker Network Plugin.

This plugin creates pure L2 LANs using bridges and veths. 

To avoid assigning any IP subnet you **MUST** use `--ipam-driver=null` when creating networks with `kathara/katharanp` plugin. Otherwise, the veth endpoint inside the container will always receive an IP address from the default pool.

## Build from the source

To build the plugin, type on terminal:
```
$ make all
```

To build the plugin for arm64, specify ARCH variable:
```
$ ARCH=arm64 make all
```

Of course, Golang should be installed (all dependencies are automatically resolved).
