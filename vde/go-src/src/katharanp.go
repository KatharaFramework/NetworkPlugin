package main

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/docker/docker/libnetwork/types"
	"github.com/docker/go-plugins-helpers/network"
	"github.com/KatharaFramework/NetworkPluginLib"
)

var (
	PLUGIN_NAME = "katharanp"
	PLUGIN_GUID = 0
)

const (
	GENERIC_OPTIONS    = "com.docker.network.generic"
	SWITCH_MODE_OPTION = "kathara.switch.mode"
	PORT_LABEL_OPTION  = "kathara.switch.label"
	PORT_VLAN_OPTION   = "kathara.switch.vlan"
	PORT_TAGGED_OPTION = "kathara.switch.tagged"
	PORT_DESCRIPTION   = "kathara"
)

type katharaEndpoint struct {
	macAddress  net.HardwareAddr
	tapIface    string
	tapIfaceIdx int
	vdeThread   uintptr
	ns          string
	// Name given to the switch port (managed switches), usually `device:ethN`
	label string
	// VLANs declared for the switch port (managed switches)
	vlans katnplib.PortVlans
	// Key of the switch port in the ports of the network, empty when the endpoint has no reserved port
	portKey string
}

// A reserved port of a managed switch
type katharaPort struct {
	number int
	// Declared VLANs which were applied to the port
	vlans katnplib.PortVlans
	// Endpoint plugged in the port, empty when there is none
	endpointID string
}

type katharaNetwork struct {
	switchName string
	mode       katnplib.SwitchMode
	ports      map[string]*katharaPort
	endpoints  map[string]*katharaEndpoint
}

func stringOption(options map[string]interface{}, name string) string {
	value, _ := options[name].(string)

	return value
}

// Return the port of a managed switch reserved for an endpoint, creating and configuring it when needed.
// A labelled endpoint finds its port again, with the VLANs set at run time, each time it joins the network:
// the declared VLANs are only applied to a new port, or when they are not the ones applied before.
func (n *katharaNetwork) reservePort(endpointID string, endpoint *katharaEndpoint) (int, error) {
	portKey := endpoint.label
	if port, ok := n.ports[portKey]; portKey == "" || (ok && port.endpointID != "" && port.endpointID != endpointID) {
		// No label, or a label already used by another endpoint: the port lives as long as the endpoint
		portKey = endpointID
	}

	port, ok := n.ports[portKey]
	if !ok {
		number, err := katnplib.CreateSwitchPort(n.switchName)
		if err != nil {
			return 0, err
		}

		port = &katharaPort{number: number}
		n.ports[portKey] = port
	}

	if !port.vlans.Equal(endpoint.vlans) {
		if err := katnplib.ConfigureSwitchPort(n.switchName, port.number, port.vlans, endpoint.vlans); err != nil {
			return 0, err
		}
		port.vlans = endpoint.vlans
	}

	port.endpointID = endpointID
	endpoint.portKey = portKey

	return port.number, nil
}

// Free the port of an endpoint which leaves a managed switch: the port of a labelled endpoint stays reserved.
func (n *katharaNetwork) releasePort(endpointID string, endpoint *katharaEndpoint) {
	port, ok := n.ports[endpoint.portKey]
	if !ok {
		return
	}

	port.endpointID = ""
	if endpoint.portKey == endpointID {
		if err := katnplib.RemoveSwitchPort(n.switchName, port.number); err != nil {
			log.Printf("Unable to remove port %d of switch %s: %v", port.number, n.switchName, err)
		}
		delete(n.ports, endpoint.portKey)
	}

	endpoint.portKey = ""
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

	genericOptions, _ := req.Options[GENERIC_OPTIONS].(map[string]interface{})
	mode, err := katnplib.ParseSwitchMode(stringOption(genericOptions, SWITCH_MODE_OPTION))
	if err != nil {
		return err
	}

	switchName, err := katnplib.CreateSwitch(req.NetworkID, mode)
	if err != nil {
		return err
	}

	katharaNetwork := &katharaNetwork{
		switchName: switchName,
		mode:       mode,
		ports:      make(map[string]*katharaPort),
		endpoints:  make(map[string]*katharaEndpoint),
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

	err := katnplib.DeleteSwitch(req.NetworkID)
	if err != nil {
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

	untaggedVlan := stringOption(req.Options, PORT_VLAN_OPTION)
	taggedVlans := stringOption(req.Options, PORT_TAGGED_OPTION)
	if (untaggedVlan != "" || taggedVlans != "") && k.networks[req.NetworkID].mode != katnplib.SwitchModeManaged {
		return nil, fmt.Errorf("VLANs can only be set on a network with %s=%s", SWITCH_MODE_OPTION, katnplib.SwitchModeManaged)
	}

	vlans, err := katnplib.ParsePortVlans(untaggedVlan, taggedVlans)
	if err != nil {
		return nil, err
	}

	intfInfo := new(network.EndpointInterface)

	if req.Options["kathara.mac_addr"] != nil {
		// Use a pre-defined MAC Address passed by the user
		intfInfo.MacAddress = req.Options["kathara.mac_addr"].(string)
	} else if req.Options["kathara.machine"] != nil && req.Options["kathara.iface"] != nil {
		// Generate the interface MAC Address by concatenating the machine name and the interface idx
		intfInfo.MacAddress = katnplib.GenerateMacAddressFromID(req.Options["kathara.machine"].(string) + "-" + req.Options["kathara.iface"].(string))
	} else if req.Interface == nil {
		// Generate the interface MAC Address by concatenating the network id and the endpoint id
		intfInfo.MacAddress = katnplib.GenerateMacAddressFromID(req.NetworkID + "-" + req.EndpointID)
	}

	parsedMac, _ := net.ParseMAC(intfInfo.MacAddress)

	endpoint := &katharaEndpoint{
		macAddress: parsedMac,
		label:      stringOption(req.Options, PORT_LABEL_OPTION),
		vlans:      vlans,
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

	delete(k.networks[req.NetworkID].endpoints, req.EndpointID)

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

	// On a hub or on a switch which is not managed, the endpoint takes the first free port
	port := 0
	description := PORT_DESCRIPTION
	if katharaNetwork.mode == katnplib.SwitchModeManaged {
		var err error
		if port, err = katharaNetwork.reservePort(req.EndpointID, endpointInfo); err != nil {
			return nil, err
		}

		if endpointInfo.label != "" {
			description += " " + endpointInfo.label
		}
	}

	tapIface, tapIfaceIdx, err := katnplib.CreateTap(endpointInfo.macAddress)
	if err != nil {
		katharaNetwork.releasePort(req.EndpointID, endpointInfo)
		return nil, err
	}

	vdeThread, err := katnplib.JoinSwitch(katharaNetwork.switchName, tapIface, port, description)
	if err != nil {
		katharaNetwork.releasePort(req.EndpointID, endpointInfo)
		return nil, err
	}

	k.networks[req.NetworkID].endpoints[req.EndpointID].tapIface = tapIface
	k.networks[req.NetworkID].endpoints[req.EndpointID].tapIfaceIdx = tapIfaceIdx
	k.networks[req.NetworkID].endpoints[req.EndpointID].vdeThread = vdeThread
	k.networks[req.NetworkID].endpoints[req.EndpointID].ns = req.SandboxKey

	resp := &network.JoinResponse{
		InterfaceName: network.InterfaceName{
			SrcName:   tapIface,
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

	endpointInfo := k.networks[req.NetworkID].endpoints[req.EndpointID]

	katnplib.LeaveSwitch(endpointInfo.vdeThread)
	k.networks[req.NetworkID].releasePort(req.EndpointID, endpointInfo)
	if err := katnplib.DeleteTap(endpointInfo.tapIface, endpointInfo.tapIfaceIdx, endpointInfo.ns); err != nil {
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
