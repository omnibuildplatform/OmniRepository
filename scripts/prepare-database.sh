#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

export MYSQL_IMAGE="mysql:8.0"

function check-prerequisites {
  echo "checking prerequisites"
  which docker >/dev/null 2>&1
  if [[ $? -ne 0 ]]; then
    echo "docker not installed, exiting."
    exit 1
  else
    echo -n "found docker, " && docker version
  fi

  echo "checking mysqladmin"
    which mysqladmin > /dev/null 2>&1
    if [[ $? -ne 0 ]]; then
      echo "mysqladmin not installed, exiting."
      exit 1
    else
      echo -n "found mysqladmin, " && mysqladmin --version
    fi
}

function mysql-cluster-up {
  echo "running up mysql local cluster"
  docker rm omni-mysql --force
  docker run -p 3306:3306 --name omni-mysql -e MYSQL_ROOT_PASSWORD=password -e MYSQL_DATABASE=omni_repository -d ${MYSQL_IMAGE}
  echo "mysql is running up with ip address $(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' omni-mysql)"
  echo "waiting mysql to be ready"
  export MYSQL_PWD=password
  while ! mysqladmin --user=root  --host "127.0.0.1" ping --silent ; do
      sleep 3
  done
  echo "mysql is ready"
  echo "Please use command : mysqladmin --user=root  --host '127.0.0.1' to connect to database"
}

echo "Preparing environment for omni-repository developing......"

check-prerequisites

mysql-cluster-up
