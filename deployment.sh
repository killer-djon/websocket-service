#!/usr/bin/env bash

DIR="$1"
BRANCH="$2"

cd "$DIR"
source <(multiwerf use 1.1 stable)

echo "RUN COMMAND: git pull origin $BRANCH"
git checkout -- .
git checkout $BRANCH
git pull origin $BRANCH
echo
echo "RUN COMMAND: sudo chown -R magerya:www-data /home/magerya/.werf"
sudo chown -R magerya:www-data /home/magerya/.werf
echo
echo "RUN COMMAND: rm -rf /home/magerya/.werf/*"
sudo rm -rf /home/magerya/.werf/*
echo
count_img=$(docker images | egrep "werf-.*websocket|.*[^_]websocket" | awk '{print $3}' | wc -l);
if [ $count_img -gt 0 ]; then
  echo "RUN COMMAND: docker images | egrep \"werf-.*websocket|.*[^_]websocket\" | awk '{print $3}' | xargs docker image rm -f"
  docker images | egrep "werf-.*websocket|.*[^_]websocket" | awk '{print $3}' | xargs docker image rm -f
fi
echo
echo "RUN: local storage deployment "
werf build --stages-storage :local
echo
echo "RUN COMMAND: werf build-and-publish --stages-storage :local --tag-git-branch werf --images-repo registry.magerya.ru/websocket-service"
werf build-and-publish --stages-storage :local --tag-git-branch $BRANCH --images-repo registry.magerya.ru/websocket-service
echo
echo "RUN COMMAND: werf converge --stages-storage :local --env werf --namespace default --images-repo registry.magerya.ru/websocket-service"
werf converge --stages-storage :local --env werf --namespace default --images-repo registry.magerya.ru/websocket-service
echo
echo "RUN COMMAND: werf cleanup --stages-storage :local --images-repo registry.magerya.ru/websocket-service"
werf cleanup --stages-storage :local --images-repo registry.magerya.ru/websocket-service
echo
