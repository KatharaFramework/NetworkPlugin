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
)

const (
	switchPrefix    	= "kt"
	switchLen   		= 12
	switchNumIfaces		= 65535
	pluginPath			= "/hosttmp/katharanp/"
)

func getSwitchName(netID string) string {
	return switchPrefix + "-" + netID[:switchLen]
}

func getSwitchPaths(name string) (string, string, string) {
	switchPath := pluginPath + name + "/"
	ctlFilePath := switchPath + "ctl"
	pidFilePath := switchPath + "pid"

	return switchPath, ctlFilePath, pidFilePath
}

func switchExists(name string) bool {
	_, _, pidFilePath := getSwitchPaths(name)
	_, err := os.Stat(pidFilePath)
	
	return err == nil
}

func CreateSwitch(netID string) (string, error) {
	switchName := getSwitchName(netID)

	exists := switchExists(switchName)
	if exists {
		return "", fmt.Errorf("switch %s already exists", switchName)
	}

	switchPath, ctlFilePath, pidFilePath := getSwitchPaths(switchName)
	os.MkdirAll(switchPath, os.ModePerm)
	cmd := exec.Command("vde_switch", "-n", strconv.Itoa(switchNumIfaces), "-x", "-d", "-s", ctlFilePath, "-p", pidFilePath)
	err := cmd.Run()
	if err != nil {
		return "", err
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

func JoinSwitch(switchName string, interfaceName string) (uintptr, error) {
	exists := switchExists(switchName)
	if !exists {
		return 0, fmt.Errorf("switch %s does not exist", switchName)
	}

	_, ctlFilePath, _ := getSwitchPaths(switchName)
	vdeThread := uintptr(C.vde_tap_plug(C.CString(interfaceName), C.CString(ctlFilePath)))
	if vdeThread == 0 {
		return 0, fmt.Errorf("unable to attack interface %s to switch %s", interfaceName, switchName)
	}

	return vdeThread, nil
}

func LeaveSwitch(vdeThread uintptr) {
	C.vde_tap_unplug(C.uintptr_t(vdeThread));
}
