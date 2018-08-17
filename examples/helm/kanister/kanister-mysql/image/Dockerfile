FROM phusion/baseimage:0.9.19
MAINTAINER "Tom Manville <tom@kasten.io>"

USER root

ADD . /kanister

RUN /kanister/install.sh && rm -rf /kanister && rm -rf /tmp && mkdir /tmp
