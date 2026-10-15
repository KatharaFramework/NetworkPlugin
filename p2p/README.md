# Kathara Network Plugin (Point-to-Point veth pairs)

## How does it work?

<p align="center">
    <img src="/images/p2p.PNG" alt="Kathara Network Plugin (Point-to-Point)" width="450" />
</p>

The Point-to-Point network plugin only supports two endpoints per network. When Docker creates the network, nothing is performed on the host (only metadata are stored). When the first container is added to the network, a veth pair is created, with one endpoint moved into the container's network namespace and the other on the host (its identity is stored in the metadata). When the second container is added, the second veth endpoint is moved to the container's network namespace. If a third container is added, the plugin returns an error.

## Advantages
- Provides the best performance when the networks are mainly point-to-point (for example, data center topologies).

## Disadvantages
- Only two endpoints are supported per network, it is basically a wire between two containers.
- No support for physical interfaces attached to containers.
- If a container dies without a clean disconnect (for example, a crash that destroys its network namespace), the kernel deletes the whole pair. The other container loses its interface, and the network has to be recreated. 

## `kathara/katharanp_p2p` Standalone Mode

It is possible to leverage on `kathara/katharanp_p2p` as a standalone Docker Network Plugin, in order to create pure L2 networks.

To create a network, type the following command:
```bash
docker network create --driver=kathara/katharanp_p2p:amd64 --ipam-driver=null l2net
# or
docker network create --driver=kathara/katharanp_p2p:arm64 --ipam-driver=null l2net
```

To avoid assigning any IP subnet you **MUST** use `--ipam-driver=null` when creating networks with Docker plugin. Otherwise, the endpoint inside the container will always receive an IP address from the default pool.