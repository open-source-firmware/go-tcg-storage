# sedlockctl

Used to operate SED locking from a Linux shell.
```
Usage: sedlockctl --device=STRING command

Go SEDlock control (temporary name)

Flags:
  -h, --help                  Show context-sensitive help.
  -d, --device=STRING         Path to SED device (e.g. /dev/nvme0)
      --sidpin=STRING
      --sidpinmsid
      --sidhash=STRING
  -u, --user=STRING
  -p, --password=STRING
  -h, --hash="sedutil-dta"

Commands:
  list          List all ranges (default)
  lock-all      Locks all ranges completely
  unlock-all    Unlocks all ranges completely
  mbrdone       Sets the MBRDone property (hide/show Shadow MBR)
  read-mbr      Prints the binary data in the MBR area
```

Example:

```
$ sudo target/sedlockctl --password debug /dev/sdd list
Range   0: whole disk [write locked] [read locked] [global]
Range   1: disabled
Range   2: disabled
Range   3: disabled
Range   4: disabled
Range   5: disabled
Range   6: disabled
Range   7: disabled
Range   8: disabled
$ sudo fdisk -l /dev/sdd
fdisk: cannot open /dev/sdd: Input/output error
$ sudo target/sedlockctl --password debug /dev/sdd unlock-all
$ sudo target/sedlockctl --password debug /dev/sdd list
Range   0: whole disk [global]
Range   1: disabled
Range   2: disabled
Range   3: disabled
Range   4: disabled
Range   5: disabled
Range   6: disabled
Range   7: disabled
Range   8: disabled
$ sudo fdisk -l /dev/sdd
Disk /dev/sdd: 465.76 GiB, 500107862016 bytes, 976773168 sectors
Disk model: Samsung SSD 860
Units: sectors of 1 * 512 = 512 bytes
Sector size (logical/physical): 512 bytes / 512 bytes
I/O size (minimum/optimal): 512 bytes / 512 bytes
```
