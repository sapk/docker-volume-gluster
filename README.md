# docker-volume-gluster [![License](https://img.shields.io/badge/license-MIT-red.svg)](https://github.com/sapk/docker-volume-gluster/blob/master/LICENSE) ![Project Status](http://img.shields.io/badge/status-alpha-red.svg)
[![GitHub release](https://img.shields.io/github/release/sapk/docker-volume-gluster.svg)](https://github.com/sapk/docker-volume-gluster/releases) [![Go Report Card](https://goreportcard.com/badge/github.com/sapk/docker-volume-gluster)](https://goreportcard.com/report/github.com/sapk/docker-volume-gluster)
[![codecov](https://codecov.io/gh/sapk/docker-volume-gluster/branch/master/graph/badge.svg)](https://codecov.io/gh/sapk/docker-volume-gluster)
 master : [![Travis master](https://api.travis-ci.org/sapk/docker-volume-gluster.svg?branch=master)](https://travis-ci.org/sapk/docker-volume-gluster) develop : [![Travis develop](https://api.travis-ci.org/sapk/docker-volume-gluster.svg?branch=develop)](https://travis-ci.org/sapk/docker-volume-gluster)

Use GlusterFS as a backend for docker volume

Status : **proof of concept (working)**

Dedends on GlusterFS (so fuse indirectly)

## Build
```
make
```

## Start daemon
```
./docker-volume-gluster daemon
OR in a docker container
docker run -d --device=/dev/fuse:/dev/fuse --cap-add=SYS_ADMIN --cap-add=MKNOD  -v /run/docker/plugins:/run/docker/plugins -v /var/lib/docker-volumes/gluster:/var/lib/docker-volumes/gluster:shared sapk/docker-volume-gluster
```

For more advance params : ```./docker-volume-gluster --help OR ./docker-volume-gluster daemon --help```
```
Run listening volume drive deamon to listen for mount request

Usage:
  docker-volume-gluster daemon [flags]

Flags:
  -o, --fuse-opts string   Fuse options to use moint point (default "big_writes,allow_other,auto_cache")

Global Flags:
  -b, --basedir string   Mounted volume base directory (default "/var/lib/docker-volumes/gluster")
  -v, --verbose          Turns on verbose logging
```

## Create and Mount volume
```
docker volume create --driver gluster --opt voluri="<volumeserver>:<volumename>" --name test
docker run -v test:/mnt --rm -ti ubuntu
```

## Docker plugin (New)
```
docker plugin install sapk/plugin-gluster
docker volume create --driver sapk/plugin-gluster --opt voluri="<volumeserver>:<volumename>" --name test
docker run -v test:/mnt --rm -ti ubuntu
```



## Docker-compose
```
volumes:
  some_vol:
    driver: sapk/plugin-gluster
    driver_opts:
      voluri: "<volumeserver>:<volumename>"
```

## Inspired from :
 - https://github.com/ContainX/docker-volume-netshare/
 - https://github.com/vieux/docker-volume-sshfs/
 - https://github.com/sapk/docker-volume-gvfs
 - https://github.com/calavera/docker-volume-glusterfs
 - https://github.com/codedellemc/rexray
