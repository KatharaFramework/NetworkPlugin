package main

import (
	"log"
	"net"
	"sync"

	"github.com/docker/go-plugins-helpers/network"
	"github.com/docker/libnetwork/types"
)

var (
	PLUGIN_NAME = "katharanp"
	PLUGIN_GUID = 0
)

type katharaEndpoint struct {
	macAddress  net.HardwareAddr
	vethInside  string
	vethOutside string
}

type katharaNetwork struct {
	bridgeName string
	endpoints  map[string]*katharaEndpoint
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

	bridgeName, err := createBridge(req.NetworkID)
	if err != nil {
		return err
	}

	katharaNetwork := &katharaNetwork{
		bridgeName: bridgeName,
		endpoints:  make(map[string]*katharaEndpoint),
	}

	k.networks[req.NetworkID] = katharaNetwork

	return nil
}

func (k *KatharaNetworkPlugin) DeleteNetwork(req *network.DeleteNetworkRequest) error {
	log.Printf("Received DeleteNetwork req:\n%+v\n", req)

	err := deleteBridge(req.NetworkID)
	if err != nil {
		return err
	}

	k.Lock()
	defer k.Unlock()

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

	intfInfo := new(network.EndpointInterface)

	if req.Interface == nil {
		intfInfo.MacAddress = generateMacAddress(req.NetworkID, req.EndpointID)
	}

	parsedMac, _ := net.ParseMAC(intfInfo.MacAddress)

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

	delete(k.networks[req.NetworkID].endpoints, req.EndpointID)

	return nil
}

func (k *KatharaNetworkPlugin) EndpointInfo(req *network.InfoRequest) (*network.InfoResponse, error) {
	log.Printf("Received EndpointOperInfo req:\n%+v\n", req)

	endpointInfo := k.networks[req.NetworkID].endpoints[req.EndpointID]
	value := make(map[string]string)

	value["ip_address"] = ""
	value["mac_address"] = endpointInfo.macAddress.String()
	value["veth_outside"] = endpointInfo.vethOutside

	resp := &network.InfoResponse{
		Value: value,
	}
	
	return resp, nil
}

func (k *KatharaNetworkPlugin) Join(req *network.JoinRequest) (*network.JoinResponse, error) {
	log.Printf("Received Join req:\n%+v\n", req)

	k.Lock()
	defer k.Unlock()

	endpointInfo := k.networks[req.NetworkID].endpoints[req.EndpointID]
	vethInside, vethOutside, err := createVethPair(endpointInfo.macAddress)
	if err != nil {
		return nil, err
	}

	if err := attachInterfaceToBridge(k.networks[req.NetworkID].bridgeName, vethOutside); err != nil {
		return nil, err
	}

	k.networks[req.NetworkID].endpoints[req.EndpointID].vethInside = vethInside
	k.networks[req.NetworkID].endpoints[req.EndpointID].vethOutside = vethOutside

	resp := &network.JoinResponse{
		InterfaceName: network.InterfaceName{
			SrcName:   vethInside,
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

	endpointInfo := k.networks[req.NetworkID].endpoints[req.EndpointID]

	if err := deleteVethPair(endpointInfo.vethOutside); err != nil {
		return err
	}

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
