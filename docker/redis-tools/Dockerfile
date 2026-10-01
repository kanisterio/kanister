FROM redis:8.10.1@sha256:8a1efc5f479551822b47424ccae982026b633f28818eab0387348120a61e10e2

RUN apt update
RUN apt install -y npm bash curl redis-tools
RUN curl -fsSL https://deb.nodesource.com/setup_current.x | bash - && apt install -y nodejs
RUN npm install redis-dump -g

RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
