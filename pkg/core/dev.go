package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/open-source-firmware/go-tcg-storage/pkg/core/feature"
	"github.com/open-source-firmware/go-tcg-storage/pkg/drive"
)

// Core holds the device interface to access IFSend/IFReceive functions as well as disk information
// obtained by the Identify and Discovery functions. This struct shall be use to interface the library
type Core struct {
	drive.DriveIntf
	DiskInfo
}

func NewCore(device string) (*Core, error) {
	d, err := drive.Open(device)
	if err != nil {
		return nil, fmt.Errorf("open device %s failed: %v", device, err)
	}
	ident, err := d.Identify()
	if err != nil {
		return nil, fmt.Errorf("identify device %s failed: %v", device, err)
	}
	c := &Core{
		DriveIntf: d,
		DiskInfo: DiskInfo{
			Identity:        ident,
			Level0Discovery: &Level0Discovery{},
		},
	}
	if err := c.Discovery0(); err != nil {
		return nil, err
	}
	return c, nil
}

// diskInfo holds information obtained by Discovery0 and Identify functions.
type DiskInfo struct {
	*Level0Discovery
	*drive.Identity
}

// Level0Discovery structure as described in TCG Storage Architecture Core Spec v2.01 rev 1.00
// (missing data length field, which is only required for parsing)
type Level0Discovery struct {
	MajorVersion                   int
	MinorVersion                   int
	Vendor                         [32]byte
	TPer                           *feature.TPer
	Locking                        *feature.Locking
	Geometry                       *feature.Geometry
	SecureMsg                      *feature.SecureMsg
	Enterprise                     *feature.Enterprise
	OpalV1                         *feature.OpalV1
	SingleUser                     *feature.SingleUser
	DataStore                      *feature.DataStore
	OpalV2                         *feature.OpalV2
	Opalite                        *feature.Opalite
	PyriteV1                       *feature.PyriteV1
	PyriteV2                       *feature.PyriteV2
	RubyV1                         *feature.RubyV1
	LockingLBA                     *feature.LockingLBA
	BlockSID                       *feature.BlockSID
	NamespaceLocking               *feature.NamespaceLocking
	DataRemoval                    *feature.DataRemoval
	NamespaceGeometry              *feature.NamespaceGeometry
	ShadowMBRForMultipleNamespaces *feature.ShadowMBRForMultipleNamespaces
	SeagatePorts                   *feature.SeagatePorts
	UnknownFeatures                []uint16
}

// Perform a Level 0 SSC Discovery.
func (d *Core) Discovery0() error {
	d0raw := make([]byte, 2048)
	if err := d.IFRecv(drive.SecurityProtocolTCGManagement, uint16(ComIDDiscoveryL0), &d0raw); err != nil {
		if err == drive.ErrNotSupported {
			return ErrNotSupported
		}
		return err
	}
	d0 := &Level0Discovery{}
	d0buf := bytes.NewBuffer(d0raw)
	d0hdr := struct {
		Size   uint32
		Major  uint16
		Minor  uint16
		_      [8]byte
		Vendor [32]byte
	}{}
	if err := binary.Read(d0buf, binary.BigEndian, &d0hdr); err != nil {
		return fmt.Errorf("failed to parse Level 0 discovery: %v", err)
	}
	if d0hdr.Size == 0 {
		return ErrNotSupported
	}
	d0.MajorVersion = int(d0hdr.Major)
	d0.MinorVersion = int(d0hdr.Minor)
	copy(d0.Vendor[:], d0hdr.Vendor[:])

	fsize := int(d0hdr.Size) - binary.Size(d0hdr) + 4
	for fsize > 0 {
		fhdr := struct {
			Code    feature.FeatureCode
			Version uint8
			Size    uint8
		}{}
		if err := binary.Read(d0buf, binary.BigEndian, &fhdr); err != nil {
			return fmt.Errorf("failed to parse feature header: %v", err)
		}
		frdr := io.LimitReader(d0buf, int64(fhdr.Size))
		var err error
		switch fhdr.Code {
		case feature.CodeTPer:
			d0.TPer, err = feature.ReadTPerFeature(frdr)
		case feature.CodeLocking:
			d0.Locking, err = feature.ReadLockingFeature(frdr)
		case feature.CodeGeometry:
			d0.Geometry, err = feature.ReadGeometryFeature(frdr)
		case feature.CodeSecureMsg:
			d0.SecureMsg, err = feature.ReadSecureMsgFeature(frdr)
		case feature.CodeEnterprise:
			d0.Enterprise, err = feature.ReadEnterpriseFeature(frdr)
		case feature.CodeOpalV1:
			d0.OpalV1, err = feature.ReadOpalV1Feature(frdr)
		case feature.CodeSingleUser:
			d0.SingleUser, err = feature.ReadSingleUserFeature(frdr)
		case feature.CodeDataStore:
			d0.DataStore, err = feature.ReadDataStoreFeature(frdr)
		case feature.CodeOpalV2:
			d0.OpalV2, err = feature.ReadOpalV2Feature(frdr)
		case feature.CodeOpalite:
			d0.Opalite, err = feature.ReadOpaliteFeature(frdr)
		case feature.CodePyriteV1:
			d0.PyriteV1, err = feature.ReadPyriteV1Feature(frdr)
		case feature.CodePyriteV2:
			d0.PyriteV2, err = feature.ReadPyriteV2Feature(frdr)
		case feature.CodeRubyV1:
			d0.RubyV1, err = feature.ReadRubyV1Feature(frdr)
		case feature.CodeLockingLBA:
			d0.LockingLBA, err = feature.ReadLockingLBAFeature(frdr)
		case feature.CodeBlockSID:
			d0.BlockSID, err = feature.ReadBlockSIDFeature(frdr)
		case feature.CodeNamespaceLocking:
			d0.NamespaceLocking, err = feature.ReadNamespaceLockingFeature(frdr)
		case feature.CodeDataRemoval:
			d0.DataRemoval, err = feature.ReadDataRemovalFeature(frdr)
		case feature.CodeNamespaceGeometry:
			d0.NamespaceGeometry, err = feature.ReadNamespaceGeometryFeature(frdr)
		case feature.CodeShadowMBRForMultipleNamespaces:
			d0.ShadowMBRForMultipleNamespaces, err = feature.ReadShadowMBRForMultipleNamespacesFeature(frdr)
		case feature.CodeSeagatePorts:
			d0.SeagatePorts, err = feature.ReadSeagatePorts(frdr)
		default:
			// Unsupported feature
			d0.UnknownFeatures = append(d0.UnknownFeatures, uint16(fhdr.Code))
		}
		if err != nil {
			return err
		}
		if _, err := io.CopyN(io.Discard, frdr, int64(fhdr.Size)); err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		fsize -= binary.Size(fhdr) + int(fhdr.Size)
	}
	d.DiskInfo.Level0Discovery = d0
	return nil
}

func (c *Core) Close() error {
	return c.DriveIntf.Close()
}
