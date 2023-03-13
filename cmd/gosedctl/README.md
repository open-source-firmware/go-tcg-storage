# Go self encrypting device control (gosedctl)

This tool provides the following functionalities:
- initial-setup:
    - Claims a given device with a given password
- load-pba:
    - Loads a Pre-Boot-Authentication image to the given device (assumed initial-setup ran first)
- revert-noerase
  - Revert the LockingSP and keep the encryption key
- revert-tper
  - Revert the tper and reset the hard drive to factory state (CAUTION: lose access to data)

## Build
Assumed path is the main folder of the repository
```
go build ./cmd/gosedctl
```

## Usage
Initial-setup
```
sudo ./gosedctl initial-setup -d /dev/<device> -p <password>
```
Load-PBA
```
sudo ./gosedctl load-pba -d /dev/<device> -p <password> -i <path/to/image>
```

## Command documentation - OPAL SSC
initial-setup
```
gosedctl initial-setup --device=STRING

Take ownership of a given device

Flags:
  -h, --help                  Show context-sensitive help.

  -d, --device=STRING         Path to SED device (e.g. /dev/nvme0)
  -p, --password=STRING
```
load-pba
```
gosedctl load-pba --device=STRING --password=STRING --path=STRING

Load PBA image to shadow MBR

Flags:
  -h, --help               Show context-sensitive help.

  -d, --device=STRING      Path to SED device (e.g. /dev/nvme0)
  -p, --password=STRING
  -i, --path=STRING        Path to PBA image
```

## Command documentation - Enterprise SSC
initial-setup-enterprise:
```
gosedctl initial-setup-enterprise --device=STRING --sid-password=STRING --band-master-0-pw=STRING --erase-master-pw=STRING

Take ownership of a given Enterprise SSC device

Flags:
  -h, --help                       Show context-sensitive help.

  -d, --device=STRING              Path to SED device (e.g. /dev/nvme0)
  -p, --sid-password=STRING        New password for SID authority
  -b, --band-master-0-pw=STRING    Password for BandMaster0 authority for configuration, lock and unlock operations.
  -e, --erase-master-pw=STRING     Password for EraseMaster authority for erase operations of ranges.
```

revert-enterprise:
```
gosedctl revert-enterprise --device=STRING --sid-password=STRING --erase-password=STRING

delete after use

Flags:
  -h, --help                     Show context-sensitive help.

  -d, --device=STRING            Path to SED device (e.g. /dev/nvme0)
  -p, --sid-password=STRING      Password to SID authority
  -e, --erase-password=STRING    Password to authenticate as EaseMaster
```

unlock-enterprise:
```
gosedctl unlock-enterprise --device=STRING --band-master-pw=STRING

Unlocks global range with BandMaster0

Flags:
  -h, --help                     Show context-sensitive help.

  -d, --device=STRING            Path to SED device (e.g. /dev/nvme0)
  -b, --band-master-pw=STRING    Password for BandMaster0 authority for configuration, lock and unlock operations.
```

## Roadmap
The intent of this command is to replace all other commands functionality and provide one binary with all capabilities.

The following list gives an overview about future capabilities:
- List Ranges
- Locking and unlocking Ranges
- Set and unset MBRDone
- Set and unset MBREnable
- Probe a device for capabilities
- to be continued...