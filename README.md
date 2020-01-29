# Katharà Network Plugin

This repository contains the Golang source code for the `kathara/katharanp` Docker Network Plugin.

This plugin creates pure L2 LANs using bridges and veths. 

To avoid assigning any IP subnet you **MUST** use `--ipam-driver=null` when creating networks with `kathara/katharanp` plugin. Otherwise, the veth endpoint inside the container will always receive an IP address from the default pool.

## Build from the source

The plugin is released with two different tags, since `iptables` v1.6 and v1.8 have different requirements.

To build `iptables v1.6` version type on terminal:
```
$ make all_stretch
```

To build `iptables v1.8` version type on terminal:
```
$ make all_buster
```

Of course, Golang should be installed (all dependencies are automatically resolved).