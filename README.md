# Kathará Network Plugin

This repository contains the source code for the Docker Network Plugin.

The plugin has two different versions, which are divided in the two main folders:
- `bridge`: legacy, creates pure L2 LANs using Linux bridges and veth pairs (which is built using the `kathara/katharanp` tag).
- `vde`: creates pure L2 LANs using VDE (Virtual Distributed Ethernet) software switches and tap interfaces (which is built using the `kathara/katharanp_vde` tag).

To avoid assigning any IP subnet you **MUST** use `--ipam-driver=null` when creating networks with Docker plugin. Otherwise, the endpoint inside the container will always receive an IP address from the default pool.

## Build from the source

The plugin is compiled and released for both `amd64` and `arm64` architectures. The tag of the plugin identifies the architecture. For backward compatibility, the `latest` tag is a retag of the `amd64` version.

To build both the plugin versions, type on terminal (in the root of the project):
```
$ make all_<arch>
```

Where `<arch>` can be: `amd64` or `arm64`.

You can also build only a specific version of the plugin by entering its directory:
```
$ cd vde && make all_<arch>
```

The build process leverages on Docker, so you don't need any dependencies installed in your machine.

## Use `katharanp` without Kathará

It is possible to leverage on `katharanp` as a standalone Docker Network Plugin, in order to create pure L2 networks.

### Note for the `bridge` version
If you are using the legacy `bridge` version, using nftables as the iptables backend requires a `xtables.lock` file in order to work properly. Hence the same host lock should be shared (and hence, mounted) with the network plugin container. 

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

### Assign MAC Addresses to network interfaces

`katharanp` supports two different ways to assign a MAC Address to a network interface:
- Pass a specific MAC Address, this can be achieved by leveraging the `kathara.mac_addr` driver option while connecting a container to a network:
```bash
docker network connect --driver-opt kathara.mac_addr=aa:bb:cc:dd:ee:ff l2net container
```
- Compute a deterministic MAC Address using the container name and the interface index, this can be done with the `kathara.machine` and `kathara.iface` driver options (they are required together):
```bash
docker network connect --driver-opt kathara.machine=container --driver-opt kathara.iface=1 l2net container
```

The formula to compute the MAC address is the following:
1. Join the two strings, separating the two values by a dash, e.g., `container-1`
2. Compute the MD5 of the resulting string: `b588c219865f6fe336908e5991216b13`
3. Take the first 6 hex bytes of the string, starting from the left: `b588c219865f` -> `b5:88:c2:19:86:5f`
4. Clean the first byte to obtain a locally administered unicast MAC Address: `0xb5 | 0x02 = 0xb7 & 0xfe = 0xb6`
5. The resulting MAC Address is: `b6:88:c2:19:86:5f`

Example output from the container:
```bash
root@584e403aec5a:/# ifconfig eth1
eth1: flags=4163<UP,BROADCAST,RUNNING,MULTICAST>  mtu 1500
        ether b6:88:c2:19:86:5f  txqueuelen 1000  (Ethernet)
        RX packets 8  bytes 736 (736.0 B)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 0  bytes 0 (0.0 B)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0
```

### Troubleshooting in standalone mode (`bridge` version)
If the Docker daemon does not start properly while using the plugin with `xtables.lock` mount (e.g., "No such file or directory" error), follow these steps.

Create a dummy `xtables.lock` file: 
```bash
touch /var/run/xtables.lock
```

Start the Docker daemon and immediately disable and remove the `katharanp` plugin:
```bash
docker plugin remove -f kathara/katharanp:amd64
# or
docker plugin remove -f kathara/katharanp:arm64
```

Install it without mounting the `xtables.lock` file:
```bash
docker plugin install kathara/katharanp:amd64
# or
docker plugin install kathara/katharanp:arm64
```
