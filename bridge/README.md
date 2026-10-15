# Kathara Network Plugin (Linux bridges + veth pairs)

## How does it work?

<p align="center">
    <img src="/images/bridge-no-ext.PNG" alt="Kathara Network Plugin (Linux Bridges)" width="450" />
</p>

When Docker creates networks using the plugin, a new Linux bridge is created for each required LAN. When a container is added to this network, a veth pair is created, with one endpoint linked to the bridge and the other end moved into the container's network namespace. Some additional `iptables` rules are applied in order to forward packets from the switch (that would be otherwise dropped by Docker isolation policies).

## Advantages
- Provides better performance (since Linux bridges are managed in the kernel);
- You can directly sniff traffic from the bridge using `tcpdump` (which in some use cases is required).

## Disadvantages
- Linux bridges do not forward arbitrary L2 multicast frames (e.g., STP);
- Unlike a hub, Linux bridges filter L2 frames for attached containers based on the bridge's forwarding database (`fdb`);
- It requires additional `iptables` rules, which comes with mananaging `nftables` vs `iptables-legacy` in the code base;
- They generate undesired noise in the form of IPv6 Router solicitations ([issue](https://github.com/KatharaFramework/NetworkPlugin/issues/4));
- The MAC addresses of the veth interfaces connected to the bridge are also learned by the containers;
- You cannot use the Docker default network in your containers, otherwise packets will be captured by `iptables` rules and forwarded differently ([issue](https://github.com/KatharaFramework/Kathara/issues/211)).

## `kathara/katharanp` Standalone Mode

It is possible to leverage on `kathara/katharanp` as a standalone Docker Network Plugin, in order to create pure L2 networks.

To create a network, type the following command:
```bash
docker network create --driver=kathara/katharanp:amd64 --ipam-driver=null l2net
# or
docker network create --driver=kathara/katharanp:arm64 --ipam-driver=null l2net
```

To avoid assigning any IP subnet you **MUST** use `--ipam-driver=null` when creating networks with Docker plugin. Otherwise, the endpoint inside the container will always receive an IP address from the default pool.

### Network Options
The Linux bridge of each network can be tuned by passing options with `-o` when creating the network:

| Option                    | Description                                                                                                                                       | Default  |
|---------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------|----------|
| `kathara.ageing_time`     | Ageing time of the MAC addresses learned by the bridge, in **centiseconds** (e.g., `300` = 3 seconds). With `0`, the bridge does not learn MAC addresses and floods all frames on every port (like a hub). | `0`      |
| `kathara.group_fwd_mask`  | Bitmask of the link-local group addresses (`01:80:c2:00:00:0X`) forwarded by the bridge, where bit `X` enables forwarding of address `X`. Decimal or hex (`0x...`) values are accepted. Bits 0-2 (STP, Pause, LACP) cannot be set, as they are restricted by the kernel. | `0xfff8` |

For example, to create a network that learns MAC addresses for 3 seconds and only forwards LLDP frames (`01:80:c2:00:00:0e`, bit 14):
```bash
docker network create --driver=kathara/katharanp:amd64 --ipam-driver=null -o kathara.ageing_time=300 -o kathara.group_fwd_mask=0x4000 l2net
```

### Attach Physical Interfaces and VLANs
**NOTE**: This feature is ONLY available for Linux-based operating systems.

<p align="center">
    <img src="/images/bridge-ext.PNG" alt="Kathara Network Plugin with Physical Interfaces (Linux Bridges)" width="450" />
</p>

It is possible to attach one or more host interfaces to a L2 LAN. Interfaces can either be physical interfaces or VLAN interfaces.
To do so, the interface should be attached to the corresponding Linux bridge (setting the `master` interface). In Kathará, this operation can be automatically performed using the [`lab.ext`](https://www.kathara.org/man-pages/kathara-lab.ext.5.html) file, but it also possible to manually perform it.

First, search the name of the Linux bridge associated to the network (in this example `l2net`):
```bash
$ docker network ls
NETWORK ID     NAME      DRIVER                    SCOPE
17366ac88720   bridge    bridge                    local
46e3206edb2a   host      host                      local
795c43f8b52d   l2net     kathara/katharanp:amd64   local
2cf01a87a072   none      null                      local
```

The name of the bridge is `kt-<NETWORK ID>`, in the example `kt-795c43f8b52d`.

For example, if you want to attach the physical interface `enp0s3` to the `l2net`, type the following command:
```bash
sudo ip link set dev enp0s3 master kt-795c43f8b52d
```

Also, if you want to attach a VLAN interface with VLAN ID=10 (on top of the physical interface `enp0s3`) to the `l2net`, type the following commands:
```bash
# Create the VLAN interface
sudo ip link add link enp0s3 name enp0s3.10 type vlan id 10
# Attach the interface to the bridge
sudo ip link set dev enp0s3.10 master kt-795c43f8b52d
```

### Note about `iptables` versions
Using nftables as the `iptables` backend requires a `xtables.lock` file in order to work properly. Hence the same host lock should be shared (and hence, mounted) with the network plugin container. 

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

### Troubleshooting
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
