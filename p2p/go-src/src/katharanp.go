package main

import (
	"log"
	"net"
	"sync"

	"github.com/docker/docker/libnetwork/types"
	"github.com/docker/go-plugins-helpers/network"
)

var (
	PLUGIN_NAME = "katharanp"
	PLUGIN_GUID = 0
)

type katharaEndpoint struct {
	macAddress  net.HardwareAddr
	vethName  	string
}

type katharaNetwork struct {
	vethPair  [2]string
	endpoints map[string]*katharaEndpoint
}

/* Returns the end of the veth pair that is not assigned to any other endpoint of the network */
func (n *katharaNetwork) freeVeth(endpointID string) string {
	for _, vethName := range n.vethPair {
		used := false
		for id, ep := range n.endpoints {
			if id != endpointID && ep.vethName == vethName {
				used = true
				break
			}
		}

		if !used {
			return vethName
		}
	}

	return ""
}

type KatharaNetworkPlugin struct {
	scope    string
	networks map[string]*katharaNetwork
	sync.Mutex
}

func (k *KatharaNetworkPlugin) GetCapabilities() (*network.CapabilitiesResponse, error) {
	log.Printf("Received GetCapabilities req")

	capabilities := &network.CapabilitiesResponse{
		Scope: k.scope,
	}

	return capabilities, nil
}

func (k *KatharaNetworkPlugin) CreateNetwork(req *network.CreateNetworkRequest) error {
	log.Printf("Received CreateNetwork req:\n%+v\n", req)

	k.Lock()
	defer k.Unlock()

	if _, ok := k.networks[req.NetworkID]; ok {
		return types.ForbiddenErrorf("network %s exists", req.NetworkID)
	}

	katharaNetwork := &katharaNetwork{
		endpoints: make(map[string]*katharaEndpoint),
	}

	k.networks[req.NetworkID] = katharaNetwork

	return nil
}

func (k *KatharaNetworkPlugin) DeleteNetwork(req *network.DeleteNetworkRequest) error {
	log.Printf("Received DeleteNetwork req:\n%+v\n", req)

	k.Lock()
	defer k.Unlock()

	/* Skip if not in map */
	if _, ok := k.networks[req.NetworkID]; !ok {
		return nil
	}

	if err := deleteVethPair(k.networks[req.NetworkID].vethPair[:]...); err != nil {
		return err
	}

	delete(k.networks, req.NetworkID)

	return nil
}

func (k *KatharaNetworkPlugin) AllocateNetwork(req *network.AllocateNetworkRequest) (*network.AllocateNetworkResponse, error) {
	log.Printf("Received AllocateNetwork req:\n%+v\n", req)

	return nil, nil
}

func (k *KatharaNetworkPlugin) FreeNetwork(req *network.FreeNetworkRequest) error {
	log.Printf("Received FreeNetwork req:\n%+v\n", req)

	return nil
}

func (k *KatharaNetworkPlugin) CreateEndpoint(req *network.CreateEndpointRequest) (*network.CreateEndpointResponse, error) {
	log.Printf("Received CreateEndpoint req:\n%+v\n", req)

	k.Lock()
	defer k.Unlock()

	/* Throw error if not in map */
	if _, ok := k.networks[req.NetworkID]; !ok {
		return nil, types.ForbiddenErrorf("%s network does not exist", req.NetworkID)
	}

	if len(k.networks[req.NetworkID].endpoints) >= 2 {
		return nil, types.ForbiddenErrorf("cannot attach more than 2 endpoints to the network")
	}

	intfInfo := new(network.EndpointInterface)

	if req.Options["kathara.mac_addr"] != nil {
		// Use a pre-defined MAC Address passed by the user
		intfInfo.MacAddress = req.Options["kathara.mac_addr"].(string)
	} else if req.Options["kathara.machine"] != nil && req.Options["kathara.iface"] != nil {
		// Generate the interface MAC Address by concatenating the machine name and the interface idx
		intfInfo.MacAddress = generateMacAddressFromID(req.Options["kathara.machine"].(string) + "-" + req.Options["kathara.iface"].(string))
	} else if req.Interface == nil {
		// Generate the interface MAC Address by concatenating the network id and the endpoint id
		intfInfo.MacAddress = generateMacAddressFromID(req.NetworkID + "-" + req.EndpointID)
	}

	macAddress := intfInfo.MacAddress
	if macAddress == "" && req.Interface != nil {
		// Docker already assigned a MAC Address, keep track of it without returning it
		macAddress = req.Interface.MacAddress
	}

	var parsedMac net.HardwareAddr
	if macAddress != "" {
		var err error
		if parsedMac, err = net.ParseMAC(macAddress); err != nil {
			return nil, types.ForbiddenErrorf("invalid MAC address %s: %s", macAddress, err)
		}
	}

	endpoint := &katharaEndpoint{
		macAddress: parsedMac,
	}

	k.networks[req.NetworkID].endpoints[req.EndpointID] = endpoint

	resp := &network.CreateEndpointResponse{
		Interface: intfInfo,
	}

	return resp, nil
}

func (k *KatharaNetworkPlugin) DeleteEndpoint(req *network.DeleteEndpointRequest) error {
	log.Printf("Received DeleteEndpoint req:\n%+v\n", req)

	k.Lock()
	defer k.Unlock()

	/* Skip if not in map (both network and endpoint) */
	if _, netOk := k.networks[req.NetworkID]; !netOk {
		return nil
	}

	if _, epOk := k.networks[req.NetworkID].endpoints[req.EndpointID]; !epOk {
		return nil
	}

	katharaNetwork := k.networks[req.NetworkID]
	delete(katharaNetwork.endpoints, req.EndpointID)

	// the veth pair is shared by both endpoints, remove it only when the last one is gone
	if len(katharaNetwork.endpoints) == 0 {
		if err := deleteVethPair(katharaNetwork.vethPair[:]...); err != nil {
			return err
		}

		katharaNetwork.vethPair = [2]string{}
	}

	return nil
}

func (k *KatharaNetworkPlugin) EndpointInfo(req *network.InfoRequest) (*network.InfoResponse, error) {
	log.Printf("Received EndpointOperInfo req:\n%+v\n", req)

	k.Lock()
	defer k.Unlock()

	/* Throw error (both network and endpoint) */
	if _, netOk := k.networks[req.NetworkID]; !netOk {
		return nil, types.ForbiddenErrorf("%s network does not exist", req.NetworkID)
	}

	if _, epOk := k.networks[req.NetworkID].endpoints[req.EndpointID]; !epOk {
		return nil, types.ForbiddenErrorf("%s endpoint does not exist", req.NetworkID)
	}

	endpointInfo := k.networks[req.NetworkID].endpoints[req.EndpointID]
	value := make(map[string]string)

	value["ip_address"] = ""
	value["mac_address"] = endpointInfo.macAddress.String()
	value["veth_name"] = endpointInfo.vethName

	resp := &network.InfoResponse{
		Value: value,
	}

	return resp, nil
}

func (k *KatharaNetworkPlugin) Join(req *network.JoinRequest) (*network.JoinResponse, error) {
	log.Printf("Received Join req:\n%+v\n", req)

	k.Lock()
	defer k.Unlock()

	/* Throw error (both network and endpoint) */
	if _, netOk := k.networks[req.NetworkID]; !netOk {
		return nil, types.ForbiddenErrorf("%s network does not exist", req.NetworkID)
	}

	if _, epOk := k.networks[req.NetworkID].endpoints[req.EndpointID]; !epOk {
		return nil, types.ForbiddenErrorf("%s endpoint does not exist", req.NetworkID)
	}

	katharaNetwork := k.networks[req.NetworkID]
	endpointInfo := katharaNetwork.endpoints[req.EndpointID]

	if katharaNetwork.vethPair[0] == "" {
		veth1, veth2, err := createVethPair()
		if err != nil {
			return nil, err
		}

		katharaNetwork.vethPair = [2]string{veth1, veth2}
	}

	vethName := katharaNetwork.freeVeth(req.EndpointID)
	if vethName == "" {
		return nil, types.ForbiddenErrorf("no free veth available in %s network", req.NetworkID)
	}

	// free end must be in the host namespace to be moved into the container
	if !vethExists(vethName) {
		return nil, types.ForbiddenErrorf("veth %s of %s network does not exist anymore", vethName, req.NetworkID)
	}

	if err := assignVethMac(vethName, endpointInfo.macAddress); err != nil {
		return nil, err
	}

	endpointInfo.vethName = vethName

	resp := &network.JoinResponse{
		InterfaceName: network.InterfaceName{
			SrcName:   endpointInfo.vethName,
			DstPrefix: "eth",
		},
		DisableGatewayService: true,
	}

	return resp, nil
}

func (k *KatharaNetworkPlugin) Leave(req *network.LeaveRequest) error {
	log.Printf("Received Leave req:\n%+v\n", req)

	k.Lock()
	defer k.Unlock()

	/* Throw error (both network and endpoint) */
	if _, netOk := k.networks[req.NetworkID]; !netOk {
		return types.ForbiddenErrorf("%s network does not exist", req.NetworkID)
	}

	if _, epOk := k.networks[req.NetworkID].endpoints[req.EndpointID]; !epOk {
		return types.ForbiddenErrorf("%s endpoint does not exist", req.NetworkID)
	}

	// Docker moves the veth back to the host namespace after leave, it is removed in DeleteEndpoint
	return nil
}

func (k *KatharaNetworkPlugin) DiscoverNew(req *network.DiscoveryNotification) error {
	log.Printf("Received DiscoverNew req:\n%+v\n", req)

	return nil
}

func (k *KatharaNetworkPlugin) DiscoverDelete(req *network.DiscoveryNotification) error {
	log.Printf("Received DiscoverDelete req:\n%+v\n", req)

	return nil
}

func (k *KatharaNetworkPlugin) ProgramExternalConnectivity(req *network.ProgramExternalConnectivityRequest) error {
	log.Printf("Received ProgramExternalConnectivity req:\n%+v\n", req)

	return nil
}

func (k *KatharaNetworkPlugin) RevokeExternalConnectivity(req *network.RevokeExternalConnectivityRequest) error {
	log.Printf("Received RevokeExternalConnectivity req:\n%+v\n", req)

	return nil
}

func NewKatharaNetworkPlugin(scope string, networks map[string]*katharaNetwork) (*KatharaNetworkPlugin, error) {
	katharanp := &KatharaNetworkPlugin{
		scope:    scope,
		networks: networks,
	}

	return katharanp, nil
}

func main() {
	driver, err := NewKatharaNetworkPlugin("local", map[string]*katharaNetwork{})

	if err != nil {
		log.Fatalf("ERROR: %s init failed!", PLUGIN_NAME)
	}

	requestHandler := network.NewHandler(driver)

	if err := requestHandler.ServeUnix(PLUGIN_NAME, PLUGIN_GUID); err != nil {
		log.Fatalf("ERROR: %s init failed!", PLUGIN_NAME)
	}
}
