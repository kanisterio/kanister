#! /bin/sh

apk add --update nodejs
# install the latest version if not installed already!
npm install -g npm

npm install -g yo grunt-cli bower express

# check locations and packages are correct
which node
node -v
which npm
npm -v
npm ls -g --depth=0
npm install elasticdump -g
