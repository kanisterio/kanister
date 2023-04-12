FROM registry.access.redhat.com/ubi9/ubi:9.1.0-1646.1669627755

RUN dnf clean all && rm -rf /var/cache/dnf && dnf -y upgrade && \
	curl -LO https://fastdl.mongodb.org/mongocli/mongodb-atlas-cli_1.6.0_linux_x86_64.rpm && \
	dnf install -y mongodb-atlas-cli_1.6.0_linux_x86_64.rpm perl-Digest-SHA jq && \
	curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash && \
	mkdir -p /root/.config/atlascli/
COPY poll.sh poll.sh