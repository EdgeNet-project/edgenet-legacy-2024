# Configuring WireGuard on EdgeNet control-plane nodes

In an EdgeNet cluster, nodes can have public or private IP addresses.
To route traffic between all the nodes, WireGuard tunnels are established between every pairs of nodes[^1].

The CNI is configured to use the VPN interface for all intra-cluster (pod-to-pod) traffic:
```yaml
# kube-system/antrea-config configmap
# antrea-agent.conf
# [...]
transportInterface: edgenetmesh0
```

As such, the VPN interface `edgenetmesh0` must be present on every nodes.
For worker nodes, this is handled by the `node` service ([`network/vpn.go`](https://github.com/EdgeNet-project/node/blob/main/pkg/network/vpn.go)).
For control-plane nodes, which doesn't run this service, the interface must be setup manually.

## Setup

1. Install WireGuard following the instructions for the node operating system: https://www.wireguard.com/install/
2. Find an unused IPv4 (`10.183.0.0/20`) and IPv6 (`fdb4:ae86:ec99:4004::/64`) address: `kubectl get vpnpeer`
3. Generate a private key: `wg genkey`
4. Create the following configuration file in `/etc/wireguard/edgenetmesh0.conf`:

```ini
[Interface]
Address    = 10.183.X.X/20             # Replace with an unused IPv4 address
Address    = fdb4:ae86:ec99:4004::X/64 # Replace with an unused IPv6 address
PrivateKey = ...                       # Replace with the result of `wg genkey`
ListenPort = 51820
PostUp     = iptables  --append FORWARD --in-interface %i --jump ACCEPT
PostUp     = ip6tables --append FORWARD --in-interface %i --jump ACCEPT
PreDown    = iptables  --delete FORWARD --in-interface %i --jump ACCEPT
PreDown    = ip6tables --delete FORWARD --in-interface %i --jump ACCEPT
```

5. Enable and start the interface: `systemctl enable --now wg-quick@edgenetmesh0`
6. Create the VPN peer object:

```yaml
# kubectl apply -f peer.yaml
apiVersion: networking.edgenet.io/v1alpha
kind: VPNPeer
metadata:
  name: ... # Replace with the node name
spec:
  addressV4: ... # Replace with the edgenetmesh0 IPv4 address
  addressV6: ... # Replace with the edgenetmesh0 IPv6 address
  endpointAddress: ... # Replace with the public IP address of the node (e.g. use https://ipinfo.io)
  endpointPort: 51820
  publicKey: ... # Replace with the result of `echo "private key generated previously" | wg pubkey`
```

7. The interface peers will then be automatically updated by the `vpnpeer` agent. It can be checked manually with the `wg` command.

[^1]: Note that private-private links which requires NAT traversal are not currently supported; *private* nodes can communicate with *public* nodes, but not other *private* nodes.
