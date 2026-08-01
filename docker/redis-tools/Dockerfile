FROM redis:8.8.1@sha256:c88d347edef6249a6d2293f926f1eeb48bd40c57cbcd02c07f52e7f1fd2cb46b

RUN apt update
RUN apt install -y npm bash curl redis-tools
RUN curl -fsSL https://deb.nodesource.com/setup_current.x | bash - && apt install -y nodejs
RUN npm install redis-dump -g

RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
