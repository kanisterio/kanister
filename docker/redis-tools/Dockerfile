FROM redis:8.10.2@sha256:d5ac52db24d4e70566fe9944f22cf5bdc2bc739b05c0f426335161ea6c23f3b3

RUN apt update
RUN apt install -y npm bash curl redis-tools
RUN curl -fsSL https://deb.nodesource.com/setup_current.x | bash - && apt install -y nodejs
RUN npm install redis-dump -g

RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
