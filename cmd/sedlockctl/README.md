# sedlockctl

Used to operate SED locking from a Linux shell.
```
Usage: sedlockctl command <device>

Go SEDlock control (temporary name)

Arguments:
  <device>    Path to SED device (e.g. /dev/nvme0)

Flags:
  -h, --help               Show context-sensitive help.
      --sidpin=STRING
      --sidpinmsid
      --sidhash="dta"      Use dta (sha1) or sha512 for SID Pin hashing
  -u, --user=STRING
      --password=STRING    Authentication password ($PASS)
      --hash="dta"         Use dta (sha1) or sha512 for password hashing ($HASH)

Commands:
  <device> list          List all ranges (default)
  <device> lock-all      Locks all ranges completely
  <device> unlock-all    Unlocks all ranges completely
  <device> mbrdone       Sets the MBRDone property (hide/show Shadow MBR)
  <device> read-mbr      Prints the binary data in the MBR area
```

#### unlock-all:
```
Usage: sedlockctl <device> unlock-all --password=STRING [flags]

Unlocks all ranges completely

Flags:
  -h, --help               Show context-sensitive help.
      --sidpin=STRING
      --sidpinmsid
      --sidhash="dta"      Use dta (sha1) or sha512 for SID Pin hashing
  -u, --user=STRING
      --password=STRING    Authentication password ($PASS)
      --hash="dta"         Use dta (sha1) or sha512 for password hashing ($HASH)

  -k, --keep-mbr-done      Keep MBRDone status as is
```

#### mbrdone:
```
Usage: sedlockctl <device> mbrdone --password=STRING [flags]

Sets the MBRDone property (hide/show Shadow MBR)

Flags:
  -h, --help               Show context-sensitive help.
      --sidpin=STRING
      --sidpinmsid
      --sidhash="dta"      Use dta (sha1) or sha512 for SID Pin hashing
  -u, --user=STRING
      --password=STRING    Authentication password ($PASS)
      --hash="dta"         Use dta (sha1) or sha512 for password hashing ($HASH)

      --done               Status to set the MBRDone
```


### Example:

```
$ sudo target/sedlockctl /dev/sdd list --password debug
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
