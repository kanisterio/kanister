FROM redis:8.10.2@sha256:97ec88159e6d7cfb5ddef9015386087d58a912de80112f06ebdfa78fa5989901

RUN apt update
RUN apt install -y npm bash curl redis-tools
RUN curl -fsSL https://deb.nodesource.com/setup_current.x | bash - && apt install -y nodejs
RUN npm install redis-dump -g

RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
