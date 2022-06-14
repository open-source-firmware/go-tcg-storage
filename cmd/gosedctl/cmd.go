package main

import (
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/open-source-firmware/go-tcg-storage/pkg/core"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/table"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/uid"

	"golang.org/x/crypto/pbkdf2"
)

// context is the context struct required by kong command line parser
type context struct{}

// initialSetupCmd is the struct for the initial-setup cmd required by kong command line parser
type initialSetupCmd struct {
	Device   string `flag:"" required:"" short:"d"  help:"Path to SED device (e.g. /dev/nvme0)"`
	Password string `flag:"" optional:"" short:"p"`
}

type loadPBAImageCmd struct {
	Device   string `flag:"" required:"" short:"d"  help:"Path to SED device (e.g. /dev/nvme0)"`
	Password string `flag:"" required:"" short:"p"`
	Path     string `flag:"" required:"" short:"i" help:"Path to PBA image"`
}

type revertTPerCmd struct {
	Device   string `flag:"" required:"" short:"d"  help:"Path to SED device (e.g. /dev/nvme0)"`
	Password string `flag:"" required:"" short:"p"`
}

type revertNoeraseCmd struct {
	Device   string `flag:"" required:"" short:"d"  help:"Path to SED device (e.g. /dev/nvme0)"`
	Password string `flag:"" required:"" short:"p"`
}

// cli is the main command line interface struct required by kong command line parser
var cli struct {
	InitialSetup  initialSetupCmd  `cmd:"" help:"Take ownership of a given device"`
	LoadPBA       loadPBAImageCmd  `cmd:"" help:"Load PBA image to shadow MBR"`
	RevertNoerase revertNoeraseCmd `cmd:"" help:""`
	RevertTper    revertTPerCmd    `cmd:"" help:""`
}

// Run executes when the initial-setup command is invoked
func (t *initialSetupCmd) Run(ctx *context) error {
	fmt.Printf("Open device: %s", t.Device)
	coreObj, err := core.NewCore(t.Device)
	if err != nil {
		return fmt.Errorf("NewCore(%s) failed: %v", t.Device, err)
	}
	fmt.Println("Find ComID")
	comID, _, err := core.FindComID(coreObj.DriveIntf, coreObj.DiskInfo.Level0Discovery)
	if err != nil {
		return fmt.Errorf("FindComID() failed: %v", err)
	}
	fmt.Println("Create new ControlSession")
	cs, err := core.NewControlSession(coreObj.DriveIntf, coreObj.Level0Discovery, core.WithComID(comID))
	if err != nil {
		return fmt.Errorf("NewControllSession() failed: %v", err)
	}

	// Take Ownership
	fmt.Println("Create new Session")
	adminSession, err := cs.NewSession(uid.AdminSP)
	if err != nil {
		return fmt.Errorf("cs.NewSession() failed: %v", err)
	}

	//Get the MSID (only works if device hasnt been claimed)
	fmt.Println("Read MSID Pin")
	msid, err := table.Admin_C_PIN_MSID_GetPIN(adminSession)
	if err != nil {
		return fmt.Errorf("Admin_C_PIN_MSID_GetPin() failed: %v", err)
	}
	// According to TCG_Storage_Opal_SSC_Application_Note_1-00_1-00-Final.pdf, p. 10 we have to close the session
	// but this is not implemented. We use ThisSp_Authenticate to elevate the session directly.
	fmt.Println("Authenticate with MSID as SID Authority at AdminSP")
	if err := table.ThisSP_Authenticate(adminSession, uid.AuthoritySID, msid); err != nil {
		return fmt.Errorf("ThisSp_Authenticate failed: %v", err)
	}
	fmt.Println("Set new password")
	// Set the new SID password. Password needs to be hashed.
	// The used algorithm is the same as used in DriveTrustAlliance implementation of sedutil-cli
	serial, err := coreObj.SerialNumber()
	if err != nil {
		return fmt.Errorf("coreObj.SerialNumber() failed: %v", err)
	}
	salt := fmt.Sprintf("%-20s", serial)
	pwhash := pbkdf2.Key([]byte(t.Password), []byte(salt[:20]), 75000, 32, sha1.New)

	if err := table.Admin_C_Pin_SID_SetPIN(adminSession, pwhash); err != nil {
		return fmt.Errorf("Admin_C_PIN_SID_SetPIN() failed: %v", err)
	}

	fmt.Println("Activate LockingSP")
	// Activate LockingSP
	lcs, err := table.Admin_SP_GetLifeCycleState(adminSession, uid.LockingSP)
	if err != nil {
		return fmt.Errorf("Admin_SP_GetLifeCycleState() failed: %v", err)
	}
	if lcs != table.ManufacturedInactive {
		return fmt.Errorf("LockingSP Lifecycle state of %s, but require %s", lcs.String(), table.ManufacturedInactive)
	}
	if err := table.LockingSPActivate(adminSession); err != nil {
		return fmt.Errorf("LockingSPActivate() failed: %v", err)
	}
	adminSession.Close()

	fmt.Println("Configure LockingRange0")
	// Configure LockingRange0
	// New Session to LockingSP required
	lockingSession, err := cs.NewSession(uid.LockingSP)
	if err != nil {
		return fmt.Errorf("NewSession() to LockingSP failed: %v", err)
	}
	defer lockingSession.Close()
	// Elevate the session to Admin1 with required credentials
	if err := table.ThisSP_Authenticate(lockingSession, uid.LockingAuthorityAdmin1, pwhash); err != nil {
		return fmt.Errorf("authenticating as Admin1 failed: %v", err)
	}

	if err := table.ConfigureLockingRange(lockingSession); err != nil {
		return fmt.Errorf("ConfigureLockingRange() failed: %v", err)
	}

	// SetLockingRange0
	fmt.Println("SetMBRDone on")
	// setMBRDone 1
	state := true
	mbr := &table.MBRControl{Done: &state}
	if err := table.MBRControl_Set(lockingSession, mbr); err != nil {
		return fmt.Errorf("MBRDone failed: %v", err)
	}
	fmt.Println("SetMBREnable on")
	// setMBREnable 1
	mbr = &table.MBRControl{Enable: &state}
	if err := table.MBRControl_Set(lockingSession, mbr); err != nil {
		return fmt.Errorf("MBREnable failed: %v", err)
	}

	return nil
}

func (l *loadPBAImageCmd) Run(ctx *context) error {
	img, err := os.ReadFile(l.Path)
	if err != nil {
		return fmt.Errorf("ReadFile(l.Path) failed: %v", err)
	}

	if l.Password == "" {
		return fmt.Errorf("empty password not allowed")
	}

	coreObj, err := core.NewCore(l.Device)
	if err != nil {
		return fmt.Errorf("NewCore() failed: %v", err)
	}

	comID, _, err := core.FindComID(coreObj.DriveIntf, coreObj.DiskInfo.Level0Discovery)
	if err != nil {
		return fmt.Errorf("FindComID() failed: %v", err)
	}
	cs, err := core.NewControlSession(coreObj.DriveIntf, coreObj.Level0Discovery, core.WithComID(comID))
	if err != nil {
		return fmt.Errorf("NewControllSession() failed: %v", err)
	}

	serial, err := coreObj.SerialNumber()
	if err != nil {
		return fmt.Errorf("coreObj.SerialNumber() failed: %v", err)
	}
	salt := fmt.Sprintf("%-20s", serial)
	pwhash := pbkdf2.Key([]byte(l.Password), []byte(salt[:20]), 75000, 32, sha1.New)

	lockingSession, err := cs.NewSession(uid.LockingSP)
	if err != nil {
		return fmt.Errorf("NewSession() to LockingSP failed: %v", err)
	}
	defer lockingSession.Close()
	// Elevate the session to Admin1 with required credentials
	if err := table.ThisSP_Authenticate(lockingSession, uid.LockingAuthorityAdmin1, pwhash); err != nil {
		return fmt.Errorf("authenticating as Admin1 failed: %v", err)
	}
	if err := table.LoadPBAImage(lockingSession, img); err != nil {
		return fmt.Errorf("LoadPBAImage() failed: %v", err)
	}

	return nil
}

func (r *revertNoeraseCmd) Run(ctx *context) error {
	if r.Password == "" {
		return fmt.Errorf("empty password not allowed")
	}

	coreObj, err := core.NewCore(r.Device)
	if err != nil {
		return fmt.Errorf("NewCore() failed: %v", err)
	}

	comID, _, err := core.FindComID(coreObj.DriveIntf, coreObj.DiskInfo.Level0Discovery)
	if err != nil {
		return fmt.Errorf("FindComID() failed: %v", err)
	}
	cs, err := core.NewControlSession(coreObj.DriveIntf, coreObj.Level0Discovery, core.WithComID(comID))
	if err != nil {
		return fmt.Errorf("NewControllSession() failed: %v", err)
	}

	serial, err := coreObj.SerialNumber()
	if err != nil {
		return fmt.Errorf("coreObj.SerialNumber() failed: %v", err)
	}
	salt := fmt.Sprintf("%-20s", serial)
	pwhash := pbkdf2.Key([]byte(r.Password), []byte(salt[:20]), 75000, 32, sha1.New)

	lockingSession, err := cs.NewSession(uid.LockingSP)
	if err != nil {
		return fmt.Errorf("NewSession() to LockingSP failed: %v", err)
	}
	defer lockingSession.Close()
	// Elevate the session to Admin1 with required credentials
	if err := table.ThisSP_Authenticate(lockingSession, uid.LockingAuthorityAdmin1, pwhash); err != nil {
		return fmt.Errorf("authenticating as Admin1 failed: %v", err)
	}

	if err := table.RevertLockingSP(lockingSession, true, pwhash); err != nil {
		return fmt.Errorf("RevertLockingSP() failed: %v", err)
	}
	return nil
}

func (r *revertTPerCmd) Run(ctx *context) error {
	coreObj, err := core.NewCore(r.Device)
	if err != nil {
		return fmt.Errorf("NewCore(%s) failed: %v", r.Device, err)
	}
	comID, _, err := core.FindComID(coreObj.DriveIntf, coreObj.DiskInfo.Level0Discovery)
	if err != nil {
		return fmt.Errorf("FindComID() failed: %v", err)
	}
	cs, err := core.NewControlSession(coreObj.DriveIntf, coreObj.Level0Discovery, core.WithComID(comID))
	if err != nil {
		return fmt.Errorf("NewControllSession() failed: %v", err)
	}
	adminSession, err := cs.NewSession(uid.AdminSP)
	if err != nil {
		return fmt.Errorf("cs.NewSession() failed: %v", err)
	}
	serial, err := coreObj.SerialNumber()
	if err != nil {
		return fmt.Errorf("coreObj.SerialNumber() failed: %v", err)
	}
	salt := fmt.Sprintf("%-20s", serial)
	pwhash := pbkdf2.Key([]byte(r.Password), []byte(salt[:20]), 75000, 32, sha1.New)

	if err := table.ThisSP_Authenticate(adminSession, uid.AuthoritySID, pwhash); err != nil {
		return fmt.Errorf("authenticating as AdminSP failed: %v", err)
	}

	if err := table.RevertTPer(adminSession); err != nil {
		return fmt.Errorf("RevertTPer() failed: %v", err)
	}
	return nil
}
