package katnplib

//#cgo LDFLAGS: -lvdeplug -lpthread
//#include <vde_tap.h>
import "C"
import (
	"fmt"
	"strings"
	"strconv"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Behaviour of the VDE switch created for a network.
type SwitchMode string

const (
	// All the frames are sent to all the ports (default).
	SwitchModeHub		SwitchMode = "hub"
	// MAC learning switch, without any configuration.
	SwitchModeSwitch	SwitchMode = "switch"
	// MAC learning switch with a management socket (VLANs, port table, ...).
	SwitchModeManaged	SwitchMode = "managed"
)

const (
	switchPrefix    	= "kt"
	switchLen   		= 12
	switchNumIfaces		= 65535
)

const (
	mgmtSocketWait		= 2 * time.Second
	mgmtSocketPoll		= 20 * time.Millisecond
)

// Directory holding the switches' control sockets and pidfiles.
var pluginPath = "/hosttmp/katharanp/"

// Socket whose group and access mode are given to the management sockets:
// whoever can use Docker can manage the switches, and nobody else.
var dockerSocketPath = "/var/run/docker.sock"

func SetPluginPath(path string) {
	// The path must end with a slash.
	pluginPath = path
}

func getSwitchName(netID string) string {
	return switchPrefix + "-" + netID[:switchLen]
}

func getSwitchPaths(name string) (string, string, string) {
	switchPath := pluginPath + name + "/"
	ctlFilePath := switchPath + "ctl"
	pidFilePath := switchPath + "pid"

	return switchPath, ctlFilePath, pidFilePath
}

func getMgmtPath(name string) string {
	switchPath, _, _ := getSwitchPaths(name)

	return switchPath + "mgmt"
}

func ParseSwitchMode(value string) (SwitchMode, error) {
	switch SwitchMode(strings.ToLower(strings.TrimSpace(value))) {
	case "", SwitchModeHub:
		return SwitchModeHub, nil
	case SwitchModeSwitch:
		return SwitchModeSwitch, nil
	case SwitchModeManaged:
		return SwitchModeManaged, nil
	}

	return "", fmt.Errorf("invalid switch mode %q (expected hub, switch or managed)", value)
}

func switchArgs(mode SwitchMode, ctlFilePath string, pidFilePath string, mgmtFilePath string) []string {
	args := []string{"-n", strconv.Itoa(switchNumIfaces)}
	if mode == SwitchModeHub {
		args = append(args, "-x")
	}
	args = append(args, "-d", "-s", ctlFilePath, "-p", pidFilePath)
	if mode == SwitchModeManaged {
		args = append(args, "-M", mgmtFilePath)
	}

	return args
}

/* Give the management socket the group and the group/other access bits of the Docker socket */
func shareMgmtSocket(mgmtFilePath string) error {
	deadline := time.Now().Add(mgmtSocketWait)
	for {
		if _, err := os.Stat(mgmtFilePath); err == nil {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("management socket %s not created", mgmtFilePath)
		}
		time.Sleep(mgmtSocketPoll)
	}

	gid := -1
	mode := os.FileMode(0600)
	if info, err := os.Stat(dockerSocketPath); err == nil {
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			gid = int(stat.Gid)
			mode |= info.Mode().Perm() & 0066
		}
	}

	if err := os.Chown(mgmtFilePath, -1, gid); err != nil {
		return err
	}

	return os.Chmod(mgmtFilePath, mode)
}

func switchExists(name string) bool {
	_, _, pidFilePath := getSwitchPaths(name)
	_, err := os.Stat(pidFilePath)
	
	return err == nil
}

func CreateSwitch(netID string, mode SwitchMode) (string, error) {
	switchName := getSwitchName(netID)

	exists := switchExists(switchName)
	if exists {
		return "", fmt.Errorf("switch %s already exists", switchName)
	}

	switchPath, ctlFilePath, pidFilePath := getSwitchPaths(switchName)
	os.MkdirAll(switchPath, os.ModePerm)
	mgmtFilePath := getMgmtPath(switchName)
	cmd := exec.Command("vde_switch", switchArgs(mode, ctlFilePath, pidFilePath, mgmtFilePath)...)
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	if mode == SwitchModeManaged {
		if err := shareMgmtSocket(mgmtFilePath); err != nil {
			/* Best effort: a managed switch is useless without its management socket */
			DeleteSwitch(netID)
			return "", err
		}
	}

	return switchName, nil
}

func DeleteSwitch(netID string) error {
	switchName := getSwitchName(netID)

	exists := switchExists(switchName)
	if !exists {
		return fmt.Errorf("switch %s does not exist", switchName)
	}

	switchPath, _, pidFilePath := getSwitchPaths(switchName)
	pid, err := os.ReadFile(pidFilePath)
	if err != nil {
        return err
    }
	intPid, err := strconv.Atoi(strings.TrimSpace(string(pid)))
	if err != nil {
        return err
    }

	proc, err := os.FindProcess(intPid)
	if err != nil {
		return err
	}
    proc.Kill()

	if err := os.RemoveAll(switchPath); err != nil {
		return err
    }

	return nil
}

/* A port greater than 0 plugs the interface in that port of the switch, 0 takes the first free one */
func JoinSwitch(switchName string, interfaceName string, port int, descr string) (uintptr, error) {
	exists := switchExists(switchName)
	if !exists {
		return 0, fmt.Errorf("switch %s does not exist", switchName)
	}

	_, ctlFilePath, _ := getSwitchPaths(switchName)
	vdeThread := uintptr(C.vde_tap_plug(C.CString(interfaceName), C.CString(ctlFilePath), C.int(port), C.CString(descr)))
	if vdeThread == 0 {
		return 0, fmt.Errorf("unable to attack interface %s to switch %s", interfaceName, switchName)
	}

	return vdeThread, nil
}

func LeaveSwitch(vdeThread uintptr) {
	C.vde_tap_unplug(C.uintptr_t(vdeThread));
}
