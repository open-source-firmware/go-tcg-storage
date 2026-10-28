# Drive sub-library

This library sends security commands to storage devices.

Drives are created from transports, which execute single SCSI commands
(`SCSITransport`), NVMe admin commands (`NVMeTransport`) or ATA commands
(`ATATransport`), with `NewSCSIDrive`, `NewNVMeDrive` and `NewATADrive`. On
Linux, `Open` uses the built-in SG_IO and NVMe ioctl transports. Other
platforms are supported by providing a transport, or by implementing
`DriveIntf` directly if the platform already provides access to the security
commands.

`NewSATTransport` sends ATA commands through a `SCSITransport` with ATA
PASS-THROUGH, for SATA drives whose SCSI/ATA Translation layer does not
translate the SCSI security commands.
