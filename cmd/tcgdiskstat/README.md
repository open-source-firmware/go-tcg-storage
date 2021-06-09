# tcgdiskstat

Utility like `blkid` or `lsscsi` meant to be used to display state of
disks supporting TCG Storage standards.

It is read-only and does not authenticate or open sessions against the drive.

Example usage:

```
$ tcgdisksstat
DEVICE         MODEL                    SERIAL                 FIRMWARE   PROTOCOL   SSC        STATE
/dev/nvme0n1   Sabrent Rocket 4.0 2TB   A0D6070C1EA788206263   RKT401.3   NVMe       Pyrite 1   lP
```

You can also use the JSON output together with example `jq`:

```
# Select a specific device
$ tcgdiskstat --output json | jq '.[] | select(.Device == "/dev/nvme0n1")'

# Grab specific properties for all devices
$ tcgdiskstat --output json | jq -r '. | map(.Device, .Identity.Model, .Level0.Locking.LockingSupported)'
```
