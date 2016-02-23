// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Windows printing.
package printer

import (
	"syscall"
	"unsafe"
	"strings"
)

//go:generate go run mksyscall_windows.go -output zapi.go printer.go

type HDC syscall.Handle

type DOC_INFO_1 struct {
	DocName    *uint16
	OutputFile *uint16
	Datatype   *uint16
}

type PRINTER_INFO_5 struct {
	PrinterName              *uint16
	PortName                 *uint16
	Attributes               uint32
	DeviceNotSelectedTimeout uint32
	TransmissionRetryTimeout uint32
}

const
	GETDEFAULT_ERROR = -50

const (
	DC_PAPERS uint16 = 2
	DC_PAPERSIZE uint16 = 3
	DC_ENUMRESOLUTIONS uint16 = 13
	DC_PAPERNAMES uint16 = 16
)

const (
	DM_COPY = 2
	DM_OUT_BUFFER = DM_COPY
)

const (
	DM_ORIENTATION = 1
	DM_PAPERSIZE = 2
	DM_PAPERLENGTH = 4
	DM_PAPERWIDTH = 8
	DM_SCALE = 0x10
	DM_POSITION = 0x20
	DM_COPIES = 0x100
	DM_DEFAULTSOURCE = 0x200
	DM_PRINTQUALITY = 0x400
	DM_COLOR = 0x800
	DM_DUPLEX = 0x1000
	DM_YRESOLUTION = 0x2000
	DM_TTOPTION = 0x4000
	DM_COLLATE = 0x8000
	DM_FORMNAME = 0x10000
	DM_LOGPIXELS = 0x20000
	DM_BITSPERPEL = 0x40000
	DM_PELSWIDTH = 0x80000
	DM_PELSHEIGHT = 0x100000
	DM_DISPLAYFLAGS = 0x200000
	DM_DISPLAYFREQUENCY = 0x400000
	DM_PANNINGWIDTH = 0x08000000
	DM_PANNINGHEIGHT = 0x10000000
	DM_ICMMETHOD = 0x00800000
	DM_ICMINTENT = 0x01000000
	DM_MEDIATYPE = 0x02000000
	DM_DITHERTYPE = 0x04000000
	DM_ICCMANUFACTURER = 0x20000000
	DM_ICCMODEL = 0x40000000
)

type DEVICE_MODE struct {
	DeviceName [32]uint16
	SpecVersion uint16
	DriverVersion uint16
	Size uint16
	DriverExtra uint16
	Fields uint32
	Orientation int16
	PaperSize int16
	PaperLength int16
	PaperWidth int16
	Scale int16
	Copies int16
	DefaultSource int16
	PrintQuality int16
	Color int16
	Duplex int16
	YResolution int16
	TTOption int16
	Collate int16
	FormName [32]uint16
	LogPixels uint16
	BitsPerPel uint32
	PelsWidth uint32
	PelsHeight uint32
	DisplayFlags uint32
	DisplayFrequency uint32
	ICMMethod uint32
	ICMIntent uint32
	MediaType uint32
	DitherType uint32
	ICCManufacturer uint32
	ICCModel uint32
	PanningWidth uint32
	PanningHeight uint32
}

const (
	PRINTER_ENUM_LOCAL       = 2
	PRINTER_ENUM_CONNECTIONS = 4
)

//sys	GetDefaultPrinter(buf *uint16, bufN *uint32) (err error) = winspool.GetDefaultPrinterW
//sys	ClosePrinter(h syscall.Handle) (err error) = winspool.ClosePrinter
//sys	OpenPrinter(name *uint16, h *syscall.Handle, defaults uintptr) (err error) = winspool.OpenPrinterW
//sys	StartDocPrinter(h syscall.Handle, level uint32, docinfo *DOC_INFO_1) (err error) = winspool.StartDocPrinterW
//sys	EndDocPrinter(h syscall.Handle) (err error) = winspool.EndDocPrinter
//sys	WritePrinter(h syscall.Handle, buf *byte, bufN uint32, written *uint32) (err error) = winspool.WritePrinter
//sys	StartPagePrinter(h syscall.Handle) (err error) = winspool.StartPagePrinter
//sys	EndPagePrinter(h syscall.Handle) (err error) = winspool.EndPagePrinter
//sys	EnumPrinters(flags uint32, name *uint16, level uint32, buf *byte, bufN uint32, needed *uint32, returned *uint32) (err error) = winspool.EnumPrintersW

//sys	DeviceCapabilities(pDevice *uint16, pPort *uint16, fwCapability uint16, pOutput *uint16, pDevMode *uint16) (length uint32, err error) = winspool.DeviceCapabilitiesW
//sys	DocumentProperties(hWnd uint32, h syscall.Handle, name *uint16, bufOut *byte, bufIn *byte, mode uint32) (length int32, err error) = winspool.DocumentPropertiesW

func Default() (string, error) {
	b := make([]uint16, 3)
	n := uint32(len(b))
	err := GetDefaultPrinter(&b[0], &n)
	if err != nil {
		if err != syscall.ERROR_INSUFFICIENT_BUFFER {
			return "", err
		}
		b = make([]uint16, n)
		err = GetDefaultPrinter(&b[0], &n)
		if err != nil {
			return "", err
		}
	}
	return syscall.UTF16ToString(b), nil
}

// ReadNames return printer names on the system
func ReadNames() ([]string, error) {
	const flags = PRINTER_ENUM_LOCAL | PRINTER_ENUM_CONNECTIONS
	var needed, returned uint32
	buf := make([]byte, 1)
	err := EnumPrinters(flags, nil, 5, &buf[0], uint32(len(buf)), &needed, &returned)
	if err != nil {
		if err != syscall.ERROR_INSUFFICIENT_BUFFER {
			return nil, err
		}
		buf = make([]byte, needed)
		err = EnumPrinters(flags, nil, 5, &buf[0], uint32(len(buf)), &needed, &returned)
		if err != nil {
			return nil, err
		}
	}
	ps := (*[1024]PRINTER_INFO_5)(unsafe.Pointer(&buf[0]))[:returned]
	names := make([]string, 0, returned)
	for _, p := range ps {
		v := (*[1024]uint16)(unsafe.Pointer(p.PrinterName))[:]
		names = append(names, syscall.UTF16ToString(v))
	}
	return names, nil
}

type Printer struct {
	h syscall.Handle
}

func Open(name string) (*Printer, error) {
	var p Printer
	// TODO: implement pDefault parameter
	err := OpenPrinter(&(syscall.StringToUTF16(name))[0], &p.h, 0)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (p *Printer) StartDocument(name, datatype string) error {
	d := DOC_INFO_1{
		DocName:    &(syscall.StringToUTF16(name))[0],
		OutputFile: nil,
		Datatype:   &(syscall.StringToUTF16(datatype))[0],
	}
	return StartDocPrinter(p.h, 1, &d)
}

func (p *Printer) Write(b []byte) (int, error) {
	var written uint32
	err := WritePrinter(p.h, &b[0], uint32(len(b)), &written)
	if err != nil {
		return 0, err
	}
	return int(written), nil
}

func (p *Printer) EndDocument() error {
	return EndDocPrinter(p.h)
}

func (p *Printer) StartPage() error {
	return StartPagePrinter(p.h)
}

func (p *Printer) EndPage() error {
	return EndPagePrinter(p.h)
}

func (p *Printer) Close() error {
	return ClosePrinter(p.h)
}

func GetPrinterPort(printerName string) (string, error) {
	printerPort := "LPT1";
	const flags = PRINTER_ENUM_LOCAL | PRINTER_ENUM_CONNECTIONS
	var needed, returned uint32
	buf := make([]byte, 1)
	err := EnumPrinters(flags, nil, 5, &buf[0], uint32(len(buf)), &needed, &returned)
	if err != nil {
		if err != syscall.ERROR_INSUFFICIENT_BUFFER {
			return "", err
		}
		buf = make([]byte, needed)
		err = EnumPrinters(flags, nil, 5, &buf[0], uint32(len(buf)), &needed, &returned)
		if err != nil {
			return "", err
		}
	}
	ps := (*[1024]PRINTER_INFO_5)(unsafe.Pointer(&buf[0]))[:returned]
	for _, p := range ps {
		v := (*[1024]uint16)(unsafe.Pointer(p.PrinterName))[:]
		if strings.Compare(syscall.UTF16ToString(v), printerName) == 0 {
			v := (*[1024]uint16)(unsafe.Pointer(p.PortName))[:]
			printerPort = strings.Split(syscall.UTF16ToString(v), ",")[0]
		}
	}
	return printerPort, nil
}

func GetAllMediaNames(printerName string, port string) ([]string, error) {
	//port, err := GetPrinterPort(printerName)
	paperNameLength := 64
	res, err := DeviceCapabilities(&(syscall.StringToUTF16(printerName))[0], &(syscall.StringToUTF16(port))[0], DC_PAPERNAMES, nil, nil)
	var nReturned int = int(res)
	if err != nil || nReturned <= 0 {
		return nil, err
	}
	buf := make([]uint16, nReturned * paperNameLength)
	_, err = DeviceCapabilities(&(syscall.StringToUTF16(printerName))[0], &(syscall.StringToUTF16(port))[0], DC_PAPERNAMES, &buf[0], nil)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0)
	for i := 0; i < nReturned; i++ {
		name := syscall.UTF16ToString(buf[i * paperNameLength: (i + 1) * paperNameLength])
		names = append(names, name)
	}
	return names, err
}

type Point struct{
	X, Y uint32
}

func GetAllMediaSizes(printerName string, port string) ([]int, error) {
	//port, err := GetPrinterPort(printerName)
	res, err := DeviceCapabilities(&(syscall.StringToUTF16(printerName))[0], &(syscall.StringToUTF16(port))[0], DC_PAPERSIZE, nil, nil)
	var nPapers int = int(res)
	if err != nil || nPapers <= 0 {
		return nil, err
	}
	buf := make([]Point, nPapers)
	_, err = DeviceCapabilities(&(syscall.StringToUTF16(printerName))[0], &(syscall.StringToUTF16(port))[0], DC_PAPERSIZE, (*uint16)(unsafe.Pointer(&buf[0])), nil)
	if err != nil {
		return nil, err
	}
	mediaSizes := make([]int, nPapers * 2)
	for i, mediaID := range buf {
		mediaSizes[i * 2] = int(mediaID.X);
		mediaSizes[i * 2 + 1] = int(mediaID.Y);
	}
	return mediaSizes, nil
}

func GetAllMediaIDs(printerName string, port string) ([]int, error) {
	//port, err := GetPrinterPort(printerName)
	res, err := DeviceCapabilities(&(syscall.StringToUTF16(printerName))[0], &(syscall.StringToUTF16(port))[0], DC_PAPERS, nil, nil)
	var numSizes int = int(res)
	if err != nil || numSizes <= 0 {
		return nil, err
	}
	buf := make([]uint16, numSizes)
	_, err = DeviceCapabilities(&(syscall.StringToUTF16(printerName))[0], &(syscall.StringToUTF16(port))[0], DC_PAPERS, &buf[0], nil)
	if err != nil {
		return nil, err
	}
	mediaIDs := make([]int, numSizes)
	for i, mediaID := range buf {
		mediaIDs[i] = int(mediaID);
	}
	return mediaIDs, nil
}

func GetAllResolutions(printerName string, port string) ([]int, error) {
	//port, err := GetPrinterPort(printerName)
	res, err := DeviceCapabilities(&(syscall.StringToUTF16(printerName))[0], &(syscall.StringToUTF16(port))[0], DC_ENUMRESOLUTIONS, nil, nil)
	var nResolutions int = int(res)
	if err != nil || nResolutions <= 0 {
		return nil, err
	}
	buf := make([]uint, nResolutions * 2)
	_, err = DeviceCapabilities(&(syscall.StringToUTF16(printerName))[0], &(syscall.StringToUTF16(port))[0], DC_ENUMRESOLUTIONS, (*uint16)(unsafe.Pointer(&buf[0])), nil)
	if err != nil {
		return nil, err
	}
	resolutions := make([]int, nResolutions * 2)
	for i, resolution := range buf {
		resolutions[i] = int(resolution)
	}
	return resolutions, nil
}

func GetDefaultSettings(printerName string, port string) ([]int, error) {
	var h syscall.Handle
	err := OpenPrinter(&(syscall.StringToUTF16(printerName))[0], &h, 0)
	if err != nil {
		return nil, err
	}
	defer ClosePrinter(h)

	res, err := DocumentProperties(0, h, &(syscall.StringToUTF16(printerName))[0], nil, nil, 0)
	var needed int = int(res)
	if err != nil || needed <= 0 {
		return nil, err
	}
	buf := make([]byte, needed)
	_, err = DocumentProperties(0, h, &(syscall.StringToUTF16(printerName))[0], &buf[0], nil, DM_OUT_BUFFER)
	if err != nil {
		return nil, err
	}

	pDevMode := (*DEVICE_MODE)(unsafe.Pointer(&buf[0]))
	defIndices := make([]int, 9)
	for i, _ := range defIndices {
		defIndices[i] = GETDEFAULT_ERROR
	}

	if pDevMode.Fields & DM_PAPERSIZE > 0 {
		defIndices[0] = int(pDevMode.PaperSize)

		//port, err := GetPrinterPort(printerName)
		res, err := DeviceCapabilities(&(syscall.StringToUTF16(printerName))[0], &(syscall.StringToUTF16(port))[0], DC_PAPERS, nil, nil)
		var numSizes int = int(res)
		if err == nil && numSizes > 0 {
			papers := make([]uint16, numSizes * 2)
			_, err = DeviceCapabilities(&(syscall.StringToUTF16(printerName))[0], &(syscall.StringToUTF16(port))[0], DC_PAPERS, &papers[0], nil)
			if err != nil {

			}
			present := false
			for size := range papers {
				if size == int(pDevMode.PaperSize) {
					present = true
				}
			}
			if (!present) {
				defIndices[0] = int(papers[0])
			}
		}
	}

	if pDevMode.Fields & DM_MEDIATYPE > 0 {
		defIndices[1] = int(pDevMode.MediaType)
	}

	if pDevMode.Fields & DM_YRESOLUTION > 0 {
		defIndices[2]  = int(pDevMode.YResolution)
	}

	if pDevMode.Fields & DM_PRINTQUALITY > 0 {
		defIndices[3] = int(pDevMode.PrintQuality)
	}

	if pDevMode.Fields & DM_COPIES > 0 {
		defIndices[4] = int(pDevMode.Copies)
	}

	if pDevMode.Fields & DM_ORIENTATION > 0 {
		defIndices[5] = int(pDevMode.Orientation)
	}

	if pDevMode.Fields & DM_DUPLEX > 0 {
		defIndices[6] = int(pDevMode.Duplex)
	}

	if pDevMode.Fields & DM_COLLATE > 0 {
		defIndices[7] = int(pDevMode.Collate)
	}

	if pDevMode.Fields & DM_COLOR > 0 {
		defIndices[8] = int(pDevMode.Color)
	}

	return defIndices, nil
}