package engine

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync"
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
			log.Printf("loadPacketsFromDir: skipping %s: %v", fullPath, err)
			continue
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
			log.Printf("readAndRewriteFile: failed to rewrite packet in %s: %v", path, err)
			continue
		}
		packets = append(packets, rewritten)
	}

	return packets, nil
}

// loadRawPacketsFromPath loads PCAP packets without rewriting addresses.
func loadRawPacketsFromPath(path string) ([]packetGroup, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	if info.IsDir() {
		return loadRawPacketsFromDir(path)
	}
	packets, err := readRawFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pcap file %s: %w", path, err)
	}
	if len(packets) == 0 {
		return nil, fmt.Errorf("no packets in %s", path)
	}
	return []packetGroup{{Name: filepath.Base(path), Packets: packets}}, nil
}

func loadRawPacketsFromDir(dirPath string) ([]packetGroup, error) {
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
		packets, err := readRawFile(fullPath)
		if err != nil {
			log.Printf("loadRawPacketsFromDir: skipping %s: %v", fullPath, err)
			continue
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

// readRawFile reads a single PCAP file and returns raw packet bytes.
func readRawFile(path string) ([][]byte, error) {
	handle, err := pcap.OpenOffline(path)
	if err != nil {
		return nil, fmt.Errorf("open pcap %s: %w", path, err)
	}
	defer handle.Close()

	var packets [][]byte
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		packets = append(packets, packet.Data())
	}
	return packets, nil
}

// randomIPv4 returns a random IPv4 address.
func randomIPv4() net.IP {
	return net.IPv4(
		byte(rand.Uint32()),
		byte(rand.Uint32()),
		byte(rand.Uint32()),
		byte(rand.Uint32()),
	)
}

// randomMAC returns a random unicast, locally administered MAC address.
func randomMAC() net.HardwareAddr {
	b := make([]byte, 6)
	for i := range b {
		b[i] = byte(rand.Uint32())
	}
	// Unicast (bit 0 = 0) + locally administered (bit 1 = 1)
	b[0] = (b[0] & 0xFE) | 0x02
	return net.HardwareAddr(b)
}

// rewritePacket modifies Ethernet MACs and IPv4 addresses in-place, returns new byte slice.
func rewritePacket(raw []byte, srcMAC, dstMAC net.HardwareAddr, srcIP, dstIP net.IP) ([]byte, error) {
	packet := gopacket.NewPacket(raw, layers.LayerTypeEthernet, gopacket.Default)

	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ethLayer == nil || ipLayer == nil {
		return nil, fmt.Errorf("packet missing ethernet or ip layer")
	}

	eth, ethOk := ethLayer.(*layers.Ethernet)
	ip, ipOk := ipLayer.(*layers.IPv4)
	if !ethOk || !ipOk {
		return nil, fmt.Errorf("type assertion failed")
	}

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
	return openRawSocketByName(iface)
}

// openRawSocketByName creates an AF_PACKET raw socket bound to the named interface.
func openRawSocketByName(iface string) (int, int, error) {
	ifaceObj, err := net.InterfaceByName(iface)
	if err != nil {
		return -1, -1, fmt.Errorf("get interface %s: %w", iface, err)
	}
	return bindRawSocket(ifaceObj)
}

// bindRawSocket creates and binds an AF_PACKET raw socket to the given interface.
func bindRawSocket(ifaceObj *net.Interface) (int, int, error) {
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

var (
	ifaceCache   = map[string]string{}
	ifaceCacheMu sync.RWMutex
)

// getInterfaceByIP finds the network interface name that has the given IP address.
// Results are cached since interfaces rarely change at runtime.
func getInterfaceByIP(target net.IP) (string, error) {
	key := target.String()
	ifaceCacheMu.RLock()
	if name, ok := ifaceCache[key]; ok {
		ifaceCacheMu.RUnlock()
		return name, nil
	}
	ifaceCacheMu.RUnlock()

	name, err := findInterfaceByIP(target)
	if err != nil {
		return "", err
	}

	ifaceCacheMu.Lock()
	ifaceCache[key] = name
	ifaceCacheMu.Unlock()

	return name, nil
}

func findInterfaceByIP(target net.IP) (string, error) {
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


