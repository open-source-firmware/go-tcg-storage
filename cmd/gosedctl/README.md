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

### Commands

```
  initial-setup               Take ownership of a given OPAL SSC device
  load-pba                    Load PBA image to shadow MBR
  revert-noerase
  revert-tper
  initial-setup-enterprise    Take ownership of a given Enterprise SSC device
  revert-enterprise           delete after use
  unlock-enterprise           Unlocks global range with BandMaster0
  reset-sid                   Resets the SID PIN to MSID
```

### Command documentation - OPAL SSC
#### initial-setup
```
gosedctl initial-setup --password=STRING <device>

Take ownership of a given OPAL SSC device

Arguments:
  <device>    Path to SED device (e.g. /dev/nvme0)

Flags:
  -h, --help                  Show context-sensitive help.

      --password=STRING       SID Password ($SID_PWD)
      --hash="sedutil-dta"    Either use sedutil-dta (sha1) or sedutil-sha512 for hashing ($SID_HASH)
```
#### load-pba
```
gosedctl load-pba --password=STRING <pba-image> <device>

Load PBA image to shadow MBR

Arguments:
  <pba-image>    Path to PBA image
  <device>       Path to SED device (e.g. /dev/nvme0)

Flags:
  -h, --help                  Show context-sensitive help.

      --password=STRING       SID Password ($SID_PWD)
      --hash="sedutil-dta"    Either use sedutil-dta (sha1) or sedutil-sha512 for hashing ($SID_HASH)
```

### Command documentation - Enterprise SSC
#### initial-setup-enterprise:
```
gosedctl initial-setup-enterprise --sid-password=STRING --bandmaster-password=STRING --erase-master-password=STRING <device>

Take ownership of a given Enterprise SSC device

Arguments:
  <device>    Path to SED device (e.g. /dev/nvme0)

Flags:
  -h, --help                               Show context-sensitive help.

      --sid-password=STRING                SID Password ($SID_PWD)
      --sid-hash="sedutil-dta"             Either use sedutil-dta (sha1) or sedutil-sha512 for hashing ($SID_HASH)
      --bandmaster-password=STRING         SID Password ($BANDMASTER_PWD)
      --bandmaster-hash="sedutil-dta"      Either use sedutil-dta (sha1) or sedutil-sha512 for hashing ($BANDMASTER_HASH)
      --erase-master-password=STRING       SID Password ($ERASE_MASTER_PWD)
      --erase-master-hash="sedutil-dta"    Either use sedutil-dta (sha1) or sedutil-sha512 for hashing ($ERASE_MASTER_HASH)
```

#### revert-enterprise:
```
gosedctl revert-enterprise --sid-password=STRING --erase-password=STRING <device>

delete after use

Arguments:
  <device>    Path to SED device (e.g. /dev/nvme0)

Flags:
  -h, --help                        Show context-sensitive help.

      --sid-password=STRING         SID Password ($SID_PWD)
      --sid-hash="sedutil-dta"      Either use sedutil-dta (sha1) or sedutil-sha512 for hashing ($SID_HASH)
      --erase-password=STRING       SID Password ($ERASE_PWD)
      --erase-hash="sedutil-dta"    Either use sedutil-dta (sha1) or sedutil-sha512 for hashing ($ERASE_HASH)
```

#### unlock-enterprise:
```
gosedctl unlock-enterprise --bandmaster-password=STRING <device>

Unlocks global range with BandMaster0

Arguments:
  <device>    Path to SED device (e.g. /dev/nvme0)

Flags:
  -h, --help                             Show context-sensitive help.

      --bandmaster-password=STRING       SID Password ($BANDMASTER_PWD)
      --bandmaster-hash="sedutil-dta"    Either use sedutil-dta (sha1) or sedutil-sha512 for hashing ($BANDMASTER_HASH)
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