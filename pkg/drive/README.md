# Drive sub-library

This library sends security commands to storage devices.

Drives are created from transports, which execute single SCSI commands
(`SCSITransport`), NVMe admin commands (`NVMeTransport`) or ATA commands
(`ATATransport`), with `NewSCSIDrive`, `NewNVMeDrive` and `NewATADrive`. On
Linux, `Open` uses the built-in SG_IO and NVMe ioctl transports. Other
platforms are supported by providing a transport, or by implementing
`DriveIntf` directly if the platform already provides access to the security
commands.
