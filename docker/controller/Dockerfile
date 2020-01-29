FROM registry.access.redhat.com/ubi8/ubi-minimal:8.1
MAINTAINER Tom Manville<tom@kasten.io>
RUN microdnf install git shadow-utils && \
    microdnf clean all && \
	adduser -U kanister
USER kanister
ADD --chown=kanister:kanister controller /controller
ENTRYPOINT ["/controller"]
