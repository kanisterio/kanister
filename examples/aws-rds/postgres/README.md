## AWS RDS Postgresql

### Steps

#### Create RDS instance


#### Backup

```
kanctl create actionset --action backup -d default/pgtestapp -c dbconfig=default/dbconfig -p default/s3-profile-6hmhn -b rds-blueprint -n kasten-io
```

#### Restore

```
kanctl create actionset --action restore -c dbconfig=default/dbconfig --from backup-sn8hk -n kasten-io
```

### Delete

```
kanctl create actionset --action delete -c dbconfig=default/dbconfig --from backup-sn8hk -n kasten-io
```
