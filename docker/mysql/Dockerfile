FROM bitnami/mysql:8.0.32-debian-11-r14

# Install kopia to take backups
COPY --from=kopia/kopia:latest /usr/bin/kopia /usr/local/bin/kopia

# Install kando
ADD kando /usr/local/bin/
