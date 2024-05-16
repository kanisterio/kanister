FROM registry.access.redhat.com/ubi9/ubi-minimal:9.4-949.1714662671

RUN cat >/etc/yum.repos.d/mongodb.repo <<EOL
[mongodb-org-6.0]
name=MongoDB Repository
baseurl=https://repo.mongodb.org/yum/redhat/9Server/mongodb-org/6.0/x86_64/
gpgcheck=1
enabled=1
gpgkey=https://pgp.mongodb.com/server-6.0.asc
EOL
RUN microdnf install yum -y && yum -y update &&\
	yum install -y mongodb-atlas perl-Digest-SHA tar jq && microdnf clean all
RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
RUN mkdir -p /root/.config/atlascli/
