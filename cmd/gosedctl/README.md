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

## Command documentation
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

## Roadmap
The intent of this command is to replace all other commands functionality and provide one binary with all capabilities.

The following list gives an overview about future capabilities:
- List Ranges
- Locking and unlocking Ranges
- Set and unset MBRDone
- Set and unset MBREnable
- Probe a device for capabilities
- to be continued...