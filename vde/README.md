# Kathara Network Plugin (VDE Switches + tap interfaces)

## How does it work?

<p align="center">
    <img src="/images/vde-no-ext.PNG" alt="Kathara Network Plugin (VDE Switches)" width="450" />
</p>

When Docker creates networks using the plugin, a new VDE switch process is created for each required LAN. When a container is added to this network, a tap interface is created and moved into the container's network namespace.

## Advantages
- Forwards arbitrary L2 multicast frames (e.g., STP);
- Behaves like a hub by default, each container on the LAN receives all the L2 frames;
- Can also behave like a learning switch or like a managed switch with VLANs, see [Switch Modes](#switch-modes);
- Does not generate undesired noise (e.g., [IPv6 router solicitations](https://github.com/KatharaFramework/NetworkPlugin/issues/4));
- You can use any L3 network, even the Docker default network.

## Disadvantages
- VDE switches are less performant than Linux bridges (they are managed in userspace);
- You cannot directly `tcpdump` traffic from the bridge. The solution is to sniff traffic directly from a container interface. 

## `kathara/katharanp_vde` Standalone Mode

It is possible to leverage on `kathara/katharanp_vde` as a standalone Docker Network Plugin, in order to create pure L2 networks.

To create a network, type the following command:
```bash
docker network create --driver=kathara/katharanp_vde:amd64 --ipam-driver=null l2net
# or
docker network create --driver=kathara/katharanp_vde:arm64 --ipam-driver=null l2net
```

To avoid assigning any IP subnet you **MUST** use `--ipam-driver=null` when creating networks with Docker plugin. Otherwise, the endpoint inside the container will always receive an IP address from the default pool.

### Switch Modes

The `kathara.switch.mode` driver option selects the behaviour of the VDE switch created for a network:

| Mode | Behaviour |
|------|-----------|
| `hub` (default) | Each frame is sent to all the containers of the network. |
| `switch` | The switch learns the MAC addresses and sends a frame only to its destination. It cannot be configured. |
| `managed` | Learning switch with VLANs and a management socket. |

```bash
docker network create --driver=kathara/katharanp_vde:amd64 --ipam-driver=null -o kathara.switch.mode=switch l2net
```

**NOTE**: a VDE switch which is not a hub only accepts the frames with an 802.1Q tag on the ports which are members of their VLAN. A VLAN trunk between two containers therefore needs a `hub` network, or a `managed` network whose ports are tagged members of the VLANs (`kathara.switch.tagged`); it does not cross a `switch` network.

The plugin declares the modes it supports in its `KATHARA_SWITCH_MODES` environment variable (`docker plugin inspect -f '{{.Settings.Env}}' kathara/katharanp_vde:amd64`), so that a client can refuse a mode instead of silently getting a hub from an older version.

#### Ports of a Managed Switch

On a `managed` network, these driver options can be given when a container is connected:

| Option | Meaning |
|--------|---------|
| `kathara.switch.label` | Name of the switch port, shown in its description (`kathara <label>`). Kathará uses `<device>:eth<N>`. |
| `kathara.switch.vlan` | VLAN (1-4094) of the untagged frames of the port (access port). Default: VLAN 0, the default VLAN of the switch. |
| `kathara.switch.tagged` | VLANs whose frames are exchanged with an 802.1Q tag on the port (trunk port), separated by commas or by spaces. |

```bash
docker network create --driver=kathara/katharanp_vde:amd64 --ipam-driver=null -o kathara.switch.mode=managed l2net
docker network connect --driver-opt kathara.switch.label=pc1:eth0 --driver-opt kathara.switch.vlan=10 l2net pc1
# The docker command line splits the values of --driver-opt on commas: separate the VLANs with spaces
docker network connect --driver-opt kathara.switch.label=r1:eth0 --driver-opt "kathara.switch.tagged=10 20" l2net r1
```

Each endpoint of a managed switch is plugged into a port reserved for it. A port with a label is kept when its container leaves the network: when a container with the same label joins again (e.g. when it is restarted), it finds the same port, with the configuration made in the meantime through the management socket. The declared VLANs are applied when the port is created, and again only when they are not the ones declared before. The port of an endpoint without label is removed when the endpoint leaves.

#### Management Socket

A managed switch has a VDE management socket in `/tmp/katharanp/<SWITCH NAME>/mgmt` on the host (the switch name is `kt-<NETWORK ID>`, as explained below). It has the group and the group/other access rights of the Docker socket: the users who can use Docker can manage the switch.

```bash
$ vdeterm /tmp/katharanp/kt-795c43f8b52d/mgmt
vde$ port/print
Port 0001 untagged_vlan=0010 ACTIVE - NOT Unnamed Allocatable
 Current User: root Access Control: (User: NONE - Group: NONE)
  -- endpoint ID 0003 module unix prog   : kathara pc1:eth0 user=0 pid=45502
1000 Success

vde$ vlan/create 30
1000 Success

vde$ port/setvlan 1 30
1000 Success
```

`help` lists the commands: `port/print`, `vlan/print`, `vlan/create`, `port/setvlan` (untagged VLAN of a port), `vlan/addport` (tagged VLAN of a port), `hash/print` (MAC address table), `port/sethub`, `fstp/setfstp`... In Kathará, the `kathara switch` command and the `exec_link` API method send these commands.

### Attach Physical Interfaces and VLANs
**NOTE**: This feature is ONLY available for Linux-based operating systems.

<p align="center">
    <img src="/images/vde-ext.PNG" alt="Kathara Network Plugin with Physical Interfaces (VDE Switches)" width="450" />
</p>

It is possible to attach one or more host interfaces to a L2 LAN. Interfaces can either be physical interfaces or VLAN interfaces.
To do so, the interface should be attached to the corresponding VDE switch. The procedure is a bit more complex than the Linux bridge version, since it relies on a custom `vde_ext` util to perform the connection (installed inside the plugin container). In Kathará, this operation is automatically performed using the [`lab.ext`](https://www.kathara.org/man-pages/kathara-lab.ext.5.html) file, but it also possible to manually perform it.

First, search the name of the switch associated to the network (in this example `l2net`):
```bash
$ docker network ls
NETWORK ID     NAME      DRIVER                    SCOPE
17366ac88720   bridge    bridge                    local
46e3206edb2a   host      host                      local
795c43f8b52d   l2net     kathara/katharanp:amd64   local
2cf01a87a072   none      null                      local
```

The name of the switch is `kt-<NETWORK ID>`, in the example `kt-795c43f8b52d`.

Search the ID of the Docker plugin:
```bash
$ docker plugin inspect kathara/katharanp_vde:amd64 | jq '.[0].Id'
# or
$ docker plugin inspect kathara/katharanp_vde:arm64 | jq '.[0].Id'

"08b6413ee58d9fce9e101dcd4fb6c5ba0643c5b40861d299b503d723413fa6fd"
```

Now, we can access the plugin container using `runc`:
```bash
sudo runc --root /run/docker/runtime-runc/plugins.moby exec -t 08b6413ee58d9fce9e101dcd4fb6c5ba0643c5b40861d299b503d723413fa6fd sh
```

Inside the container, we can use the `vde_ext` util to connect the physical/VLAN interface to the VDE switch. The `vde_ext` util requires the VDE socket path (`-s`), which is located in `/hosttmp/katharanp/<SWITCH NAME>/ctl`, a file path location where to store the PID of the process (`-p`), and the interface to connect. 

For example, if you want to attach the physical interface `enp0s3` to the `l2net`, type the following command:
```bash
vde_ext -s /hosttmp/katharanp/kt-795c43f8b52d/ctl -p /hosttmp/katharanp/kt-795c43f8b52d/pid_enp0s3 enp0s3 &
```

**NOTE**: The command should be detached from the current shell (using `&`).

Also, if you want to attach a VLAN interface with VLAN ID=10 (on top of the physical interface `enp0s3`) to the `l2net`, type the following commands (directly inside the plugin container):
```bash
# Create the VLAN interface
ip link add link enp0s3 name enp0s3.10 type vlan id 10
# Attach the interface to the switch
vde_ext -s /hosttmp/katharanp/kt-795c43f8b52d/ctl -p /hosttmp/katharanp/kt-795c43f8b52d/pid_enp0s3.10 enp0s3.10 &
```

To detach it, you have to kill the `vde_ext` process (using a `SIGINT` signal for proper handling):
```bash
kill -2 $(cat /hosttmp/katharanp/kt-795c43f8b52d/pid_enp0s3.10)
```
