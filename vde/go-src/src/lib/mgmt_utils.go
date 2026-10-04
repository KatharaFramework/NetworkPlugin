package katnplib

import (
	"bytes"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"
)

const (
	mgmtPrompt		= "\nvde$ "
	mgmtDataBegin		= "0000 DATA END WITH"
	mgmtDataEnd		= "."
	mgmtSuccess		= 1000
	mgmtTimeout		= 5 * time.Second
	mgmtRemoveRetries	= 20
	mgmtRemoveDelay		= 50 * time.Millisecond

	minVlanID		= 1
	maxVlanID		= 4094
)

var (
	mgmtStatusRegexp	= regexp.MustCompile(`^(1\d{3})(?: (.*))?$`)
	mgmtPortRegexp		= regexp.MustCompile(`^Port (\d+)`)
)

/* VLANs of a switch port: the untagged one (0 is the default VLAN) and the tagged ones */
type PortVlans struct {
	Untagged	int
	Tagged		[]int
}

func (v PortVlans) Equal(other PortVlans) bool {
	if v.Untagged != other.Untagged || len(v.Tagged) != len(other.Tagged) {
		return false
	}
	for i := range v.Tagged {
		if v.Tagged[i] != other.Tagged[i] {
			return false
		}
	}

	return true
}

func (v PortVlans) hasTagged(vlan int) bool {
	for _, tagged := range v.Tagged {
		if tagged == vlan {
			return true
		}
	}

	return false
}

func ParseVlanID(value string) (int, error) {
	vlan, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || vlan < minVlanID || vlan > maxVlanID {
		return 0, fmt.Errorf("invalid VLAN ID %q (expected a number between %d and %d)", value, minVlanID, maxVlanID)
	}

	return vlan, nil
}

/* Parse the VLANs of a port: an optional untagged VLAN ID and a list of tagged VLAN IDs, separated by
   commas or by spaces (the docker command line splits the values of its driver options on commas) */
func ParsePortVlans(untagged string, tagged string) (PortVlans, error) {
	vlans := PortVlans{}

	if strings.TrimSpace(untagged) != "" {
		vlan, err := ParseVlanID(untagged)
		if err != nil {
			return PortVlans{}, err
		}
		vlans.Untagged = vlan
	}

	if strings.TrimSpace(tagged) != "" {
		isSeparator := func(r rune) bool { return r == ',' || unicode.IsSpace(r) }
		for _, item := range strings.FieldsFunc(tagged, isSeparator) {
			vlan, err := ParseVlanID(item)
			if err != nil {
				return PortVlans{}, err
			}
			if vlan == vlans.Untagged {
				return PortVlans{}, fmt.Errorf("VLAN %d cannot be both untagged and tagged", vlan)
			}
			if !vlans.hasTagged(vlan) {
				vlans.Tagged = append(vlans.Tagged, vlan)
			}
		}
		sort.Ints(vlans.Tagged)
	}

	return vlans, nil
}

/* Split the reply to a management command into its status code, its status message and its data lines */
func parseMgmtReply(reply string) (int, string, []string, error) {
	code := -1
	message := ""
	lines := []string{}
	inData := false

	for _, line := range strings.Split(reply, "\n") {
		line = strings.TrimRight(line, "\r")
		if inData {
			if line == mgmtDataEnd {
				inData = false
			} else {
				lines = append(lines, line)
			}
		} else if strings.HasPrefix(line, mgmtDataBegin) {
			inData = true
		} else if matches := mgmtStatusRegexp.FindStringSubmatch(line); matches != nil {
			code, _ = strconv.Atoi(matches[1])
			message = matches[2]
		}
	}

	if code < 0 {
		return code, "", lines, fmt.Errorf("no status in the reply of the switch: %q", reply)
	}

	return code, message, lines, nil
}

func parseCreatedPort(lines []string) (int, error) {
	for _, line := range lines {
		if matches := mgmtPortRegexp.FindStringSubmatch(line); matches != nil {
			return strconv.Atoi(matches[1])
		}
	}

	return 0, fmt.Errorf("no port number in the reply of the switch: %q", lines)
}

type mgmtSession struct {
	conn	net.Conn
}

func openMgmt(switchName string) (*mgmtSession, error) {
	conn, err := net.DialTimeout("unix", getMgmtPath(switchName), mgmtTimeout)
	if err != nil {
		return nil, err
	}

	session := &mgmtSession{conn: conn}
	/* Skip the banner */
	if _, err := session.readReply(); err != nil {
		conn.Close()
		return nil, err
	}

	return session, nil
}

func (s *mgmtSession) close() {
	s.conn.Close()
}

/* Read up to the next prompt */
func (s *mgmtSession) readReply() (string, error) {
	s.conn.SetDeadline(time.Now().Add(mgmtTimeout))

	data := []byte{}
	buffer := make([]byte, 4096)
	for !bytes.HasSuffix(data, []byte(mgmtPrompt)) {
		n, err := s.conn.Read(buffer)
		data = append(data, buffer[:n]...)
		if err != nil {
			return string(data), err
		}
	}

	return string(data[:len(data) - len(mgmtPrompt)]), nil
}

func (s *mgmtSession) exec(command string) (int, string, []string, error) {
	s.conn.SetDeadline(time.Now().Add(mgmtTimeout))
	if _, err := s.conn.Write([]byte(command + "\n")); err != nil {
		return -1, "", nil, err
	}

	reply, err := s.readReply()
	if err != nil {
		return -1, "", nil, err
	}

	return parseMgmtReply(reply)
}

/* Run a command which must succeed, the errno values listed in tolerated being accepted too */
func (s *mgmtSession) run(command string, tolerated ...syscall.Errno) ([]string, error) {
	code, message, lines, err := s.exec(command)
	if err != nil {
		return nil, fmt.Errorf("switch command `%s`: %v", command, err)
	}

	if code != mgmtSuccess {
		for _, errno := range tolerated {
			if code == mgmtSuccess + int(errno) {
				return lines, nil
			}
		}
		return nil, fmt.Errorf("switch command `%s` failed: %d %s", command, code, message)
	}

	return lines, nil
}

/* Run a command on the management socket of a managed switch */
func MgmtExec(switchName string, command string) (int, string, []string, error) {
	session, err := openMgmt(switchName)
	if err != nil {
		return -1, "", nil, err
	}
	defer session.close()

	return session.exec(command)
}

/* Reserve a port of a managed switch: it is never given to an endpoint which does not ask for it,
   and it keeps its VLANs when its endpoint leaves */
func CreateSwitchPort(switchName string) (int, error) {
	session, err := openMgmt(switchName)
	if err != nil {
		return 0, err
	}
	defer session.close()

	lines, err := session.run("port/createauto")
	if err != nil {
		return 0, err
	}

	port, err := parseCreatedPort(lines)
	if err != nil {
		return 0, err
	}

	/* The switch may give a port which was used before: start from a new one, in the default VLAN */
	if _, err := session.run(fmt.Sprintf("port/remove %d", port)); err != nil {
		return 0, err
	}
	if _, err := session.run(fmt.Sprintf("port/create %d", port)); err != nil {
		return 0, err
	}

	return port, nil
}

/* Remove a port of a managed switch, once its endpoint has left */
func RemoveSwitchPort(switchName string, port int) error {
	session, err := openMgmt(switchName)
	if err != nil {
		return err
	}
	defer session.close()

	command := fmt.Sprintf("port/remove %d", port)
	for i := 0; ; i++ {
		code, message, _, err := session.exec(command)
		if err != nil {
			return err
		}
		if code == mgmtSuccess || code == mgmtSuccess + int(syscall.ENXIO) {
			return nil
		}
		/* The switch has not seen yet that the endpoint is closed */
		if code != mgmtSuccess + int(syscall.EADDRINUSE) || i >= mgmtRemoveRetries {
			return fmt.Errorf("switch command `%s` failed: %d %s", command, code, message)
		}
		time.Sleep(mgmtRemoveDelay)
	}
}

/* Move a port of a managed switch from its previous VLANs to the new ones */
func ConfigureSwitchPort(switchName string, port int, previous PortVlans, vlans PortVlans) error {
	session, err := openMgmt(switchName)
	if err != nil {
		return err
	}
	defer session.close()

	for _, vlan := range previous.Tagged {
		if !vlans.hasTagged(vlan) {
			if _, err := session.run(fmt.Sprintf("vlan/delport %d %d", vlan, port), syscall.ENXIO); err != nil {
				return err
			}
		}
	}

	if vlans.Untagged != previous.Untagged {
		if vlans.Untagged != 0 {
			if _, err := session.run(fmt.Sprintf("vlan/create %d", vlans.Untagged), syscall.EEXIST); err != nil {
				return err
			}
		}
		if _, err := session.run(fmt.Sprintf("port/setvlan %d %d", port, vlans.Untagged)); err != nil {
			return err
		}
	}

	for _, vlan := range vlans.Tagged {
		if _, err := session.run(fmt.Sprintf("vlan/create %d", vlan), syscall.EEXIST); err != nil {
			return err
		}
		if _, err := session.run(fmt.Sprintf("vlan/addport %d %d", vlan, port)); err != nil {
			return err
		}
	}

	return nil
}
