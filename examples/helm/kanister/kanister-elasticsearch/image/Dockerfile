FROM kanisterio/kanister-tools:0.23.0
MAINTAINER Supriya Kharade <supriya@kasten.io>

ADD examples/helm/kanister/kanister-elasticsearch/image/esdump-setup.sh /esdump-setup.sh
RUN chmod +x /esdump-setup.sh && sync && /esdump-setup.sh
