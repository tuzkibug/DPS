package engine

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// packetGroup holds all packets from a single PCAP file, rewritten for target addresses.
type packetGroup struct {
	Name    string
	Packets [][]byte
}

// loadPacketsFromPath loads PCAP packets from a file or directory.
func loadPacketsFromPath(path, srcMAC, dstMAC, srcIP, dstIP string) ([]packetGroup, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	if info.IsDir() {
		return loadPacketsFromDir(path, srcMAC, dstMAC, srcIP, dstIP)
	}
	packets, err := readAndRewriteFile(path, srcMAC, dstMAC, srcIP, dstIP)
	if err != nil {
		return nil, fmt.Errorf("read pcap file %s: %w", path, err)
	}
	if len(packets) == 0 {
		return nil, fmt.Errorf("no packets in %s", path)
	}
	return []packetGroup{{Name: filepath.Base(path), Packets: packets}}, nil
}

// loadPacketsFromDir reads all .pcap/.pcapng/.cap files from a directory,
// parses each packet, rewrites addresses, and returns ordered groups.
func loadPacketsFromDir(dirPath, srcMAC, dstMAC, srcIP, dstIP string) ([]packetGroup, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dirPath, err)
	}

	var pcapFiles []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext == ".pcap" || ext == ".pcapng" || ext == ".cap" {
			pcapFiles = append(pcapFiles, e.Name())
		}
	}
	sort.Strings(pcapFiles)

	var groups []packetGroup
	for _, name := range pcapFiles {
		fullPath := filepath.Join(dirPath, name)
		packets, err := readAndRewriteFile(fullPath, srcMAC, dstMAC, srcIP, dstIP)
		if err != nil {
			continue // skip unreadable files
		}
		if len(packets) > 0 {
			groups = append(groups, packetGroup{Name: name, Packets: packets})
		}
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("no valid PCAP files found in %s", dirPath)
	}

	return groups, nil
}

// readAndRewriteFile reads a single PCAP file, rewrites each packet, returns raw bytes.
func readAndRewriteFile(path, srcMAC, dstMAC, srcIP, dstIP string) ([][]byte, error) {
	handle, err := pcap.OpenOffline(path)
	if err != nil {
		return nil, fmt.Errorf("open pcap %s: %w", path, err)
	}
	defer handle.Close()

	hwSrcMAC, _ := net.ParseMAC(srcMAC)
	hwDstMAC, _ := net.ParseMAC(dstMAC)
	ipSrc := net.ParseIP(srcIP).To4()
	ipDst := net.ParseIP(dstIP).To4()

	var packets [][]byte
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		raw := packet.Data()
		rewritten, err := rewritePacket(raw, hwSrcMAC, hwDstMAC, ipSrc, ipDst)
		if err != nil {
			continue
		}
		packets = append(packets, rewritten)
	}

	return packets, nil
}

// rewritePacket modifies Ethernet MACs and IPv4 addresses in-place, returns new byte slice.
func rewritePacket(raw []byte, srcMAC, dstMAC net.HardwareAddr, srcIP, dstIP net.IP) ([]byte, error) {
	packet := gopacket.NewPacket(raw, layers.LayerTypeEthernet, gopacket.Default)

	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ethLayer == nil || ipLayer == nil {
		return nil, fmt.Errorf("packet missing ethernet or ip layer")
	}

	eth, _ := ethLayer.(*layers.Ethernet)
	ip, _ := ipLayer.(*layers.IPv4)

	// Rewrite addresses
	copy(eth.SrcMAC, srcMAC)
	copy(eth.DstMAC, dstMAC)
	copy(ip.SrcIP, srcIP)
	copy(ip.DstIP, dstIP)

	// Fix up UDP/TCP checksum by setting network layer reference
	if t := packet.TransportLayer(); t != nil {
		switch tl := t.(type) {
		case *layers.UDP:
			tl.SetNetworkLayerForChecksum(ip)
		case *layers.TCP:
			tl.SetNetworkLayerForChecksum(ip)
		}
	}

	// Serialize with modified Ethernet + IPv4, keeping all upper layers
	buf := gopacket.NewSerializeBuffer()
	var serialLayers []gopacket.SerializableLayer
	serialLayers = append(serialLayers, eth, ip)
	for _, l := range packet.Layers() {
		switch l.LayerType() {
		case layers.LayerTypeEthernet, layers.LayerTypeIPv4:
			continue
		default:
			if s, ok := l.(gopacket.SerializableLayer); ok {
				serialLayers = append(serialLayers, s)
			} else {
				serialLayers = append(serialLayers, gopacket.Payload(l.LayerContents()))
			}
		}
	}

	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, serialLayers...); err != nil {
		return nil, fmt.Errorf("serialize layers: %w", err)
	}

	return buf.Bytes(), nil
}

// openRawSocket creates an AF_PACKET raw socket bound to the interface with the given source IP.
func openRawSocket(srcIP net.IP) (int, int, error) {
	iface, err := getInterfaceByIP(srcIP)
	if err != nil {
		return -1, -1, fmt.Errorf("find interface for %s: %w", srcIP, err)
	}

	ifaceObj, err := net.InterfaceByName(iface)
	if err != nil {
		return -1, -1, fmt.Errorf("get interface %s: %w", iface, err)
	}

	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(syscall.ETH_P_ALL)))
	if err != nil {
		return -1, -1, fmt.Errorf("create raw socket: %w", err)
	}

	addr := syscall.SockaddrLinklayer{
		Protocol: htons(syscall.ETH_P_ALL),
		Ifindex:  ifaceObj.Index,
	}
	if err := syscall.Bind(fd, &addr); err != nil {
		syscall.Close(fd)
		return -1, -1, fmt.Errorf("bind raw socket: %w", err)
	}

	return fd, ifaceObj.Index, nil
}

// getInterfaceByIP finds the network interface name that has the given IP address.
func getInterfaceByIP(target net.IP) (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			if ipNet.IP.Equal(target) {
				return iface.Name, nil
			}
		}
	}

	// Fallback: return any non-loopback interface
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}
		return iface.Name, nil
	}

	return "", fmt.Errorf("no suitable interface found")
}

// WriteRaw writes data to a raw socket fd.
func WriteRaw(fd int, data []byte) error {
	return syscall.Sendto(fd, data, 0, nil)
}

// CloseRaw closes a raw socket fd.
func CloseRaw(fd int) error {
	return syscall.Close(fd)
}

// htons converts a uint16 from host to network byte order.
func htons(v uint16) uint16 {
	return (v<<8)&0xff00 | (v>>8)&0x00ff
}


