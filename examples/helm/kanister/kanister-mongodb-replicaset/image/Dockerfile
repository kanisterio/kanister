ARG TOOLS_IMAGE
FROM registry.access.redhat.com/ubi8/ubi:8.1 as builder
RUN dnf install -y https://repo.mongodb.org/yum/redhat/8/mongodb-org/4.2/x86_64/RPMS/mongodb-org-mongos-4.2.1-1.el8.x86_64.rpm

RUN dnf install -y https://repo.mongodb.org/yum/redhat/8/mongodb-org/4.2/x86_64/RPMS/mongodb-org-server-4.2.1-1.el8.x86_64.rpm

RUN dnf install -y https://repo.mongodb.org/yum/redhat/8/mongodb-org/4.2/x86_64/RPMS/mongodb-org-shell-4.2.1-1.el8.x86_64.rpm

RUN dnf install -y https://repo.mongodb.org/yum/redhat/8/mongodb-org/4.2/x86_64/RPMS/mongodb-org-tools-4.2.1-1.el8.x86_64.rpm

RUN dnf install -y https://repo.mongodb.org/yum/redhat/8/mongodb-org/4.2/x86_64/RPMS/mongodb-org-4.2.1-1.el8.x86_64.rpm

RUN dnf install -y mongodb-org

RUN dnf install -y python3-pip

Run pip3 install awscli --upgrade --user

FROM $TOOLS_IMAGE

RUN microdnf update && microdnf install tar gzip && \
    microdnf clean all

COPY --from=builder /usr/bin/mongo* /usr/bin/
COPY --from=builder /usr/bin/python* /usr/bin/
COPY --from=builder /usr/bin/pip* /usr/bin/
COPY --from=builder /root/.local/bin /root/.local/bin

RUN export PATH=~/.local/bin:$PATH
