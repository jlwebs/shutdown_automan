//go:build windows

package service

import (
	"syscall"
	"unsafe"
)

var (
	modiphlpapi      = syscall.NewLazyDLL("iphlpapi.dll")
	procGetIfTable   = modiphlpapi.NewProc("GetIfTable")
	procGetIfTable2  = modiphlpapi.NewProc("GetIfTable2")
	procFreeMibTable = modiphlpapi.NewProc("FreeMibTable")
)

type MIB_IFROW struct {
	Name            [256]uint16
	Index           uint32
	Type            uint32
	Mtu             uint32
	Speed           uint32
	PhysAddrLen     uint32
	PhysAddr        [8]byte
	AdminStatus     uint32
	OperStatus      uint32
	LastChange      uint32
	InOctets        uint32
	InUcastPkts     uint32
	InNUcastPkts    uint32
	InDiscards      uint32
	InErrors        uint32
	InUnknownProtos uint32
	OutOctets       uint32
	OutUcastPkts    uint32
	OutNUcastPkts   uint32
	OutDiscards     uint32
	OutErrors       uint32
	OutQLen         uint32
	DescrLen        uint32
	BDescr          [256]byte
}

func GetSystemNetworkStats() (bytesRecv, bytesSent uint64, err error) {
	// Simple GetIfTable, calling it twice. First to get size.
	var size uint32 = 0
	ret, _, _ := procGetIfTable.Call(
		uintptr(0),
		uintptr(unsafe.Pointer(&size)),
		uintptr(0),
	)

	if ret == 122 { // ERROR_INSUFFICIENT_BUFFER
		buf := make([]byte, size)
		ret2, _, _ := procGetIfTable.Call(
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&size)),
			uintptr(0),
		)
		if ret2 == 0 { // NO_ERROR
			numEntries := *(*uint32)(unsafe.Pointer(&buf[0]))
			offset := unsafe.Sizeof(uint32(0))
			rowSize := unsafe.Sizeof(MIB_IFROW{})

			var totalIn, totalOut uint64
			for i := uint32(0); i < numEntries; i++ {
				// Prevent panic if size is smaller than expected
				if offset+rowSize > uintptr(len(buf)) {
					break
				}
				row := (*MIB_IFROW)(unsafe.Pointer(&buf[offset]))
				totalIn += uint64(row.InOctets)
				totalOut += uint64(row.OutOctets)
				offset += rowSize
			}
			return totalIn, totalOut, nil
		}
	}
	return 0, 0, syscall.EINVAL
}
