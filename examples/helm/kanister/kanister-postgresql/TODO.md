## Blueprint Enhancements

The current version of the blueprint is intended to be a proof of concept for
point-in-time backup and recovery for Postgres. The list below is a preliminary
set of enhancement that will be needed to make this more robust.

* **Allow recovery to a specific point in time**
* **Full-backup improvements**
    + **Snapshot-based full** - reduce full backup recovery window (fewer WALs
    to apply compared to live copy) by taking a snapshot of data volume and
    copying data files from it. Potentially use snapshot as full backup
    artifact
    + **Differential transfer and dedup** - minimize the amount that is
    transferred and stored for full backups. Consider separating this as a
    Kanister upload function
* **Data restore coordination** - Kanister function to handle replacing data
  files under the primary database container in a more robust fashion
* **Optimize WAL transfer** - tune wal-e parameters to allow for faster transfer
  of WALs
* **Backup artifact management** - use Kanister artifact mechanisms to keep
  track of full backups and associated WALs as opposed to relying on single
  WAL prefix
* **Migration support** - add support for migration which will allow for a
  hot-standby in a different cluster