#!/bin/bash
sudo bash -c 'cat <<EOT > /etc/docker/daemon.json
{
  "registry-mirrors": ["https://dockerhub.semaphoreci.com/"],
  "insecure-registries": ["172.30.0.0/16"]
}
EOT'
