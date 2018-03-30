# docker-volume-gluster [![License](https://img.shields.io/badge/license-MIT-red.svg)](https://github.com/sapk/docker-volume-gluster/blob/master/LICENSE) ![Project Status](http://img.shields.io/badge/status-beta-orange.svg)
[![GitHub release](https://img.shields.io/github/release/sapk/docker-volume-gluster.svg)](https://github.com/sapk/docker-volume-gluster/releases) [![Go Report Card](https://goreportcard.com/badge/github.com/sapk/docker-volume-gluster)](https://goreportcard.com/report/github.com/sapk/docker-volume-gluster)
[![codecov](https://codecov.io/gh/sapk/docker-volume-gluster/branch/master/graph/badge.svg)](https://codecov.io/gh/sapk/docker-volume-gluster)
 master : [![Travis master](https://api.travis-ci.org/sapk/docker-volume-gluster.svg?branch=master)](https://travis-ci.org/sapk/docker-volume-gluster) develop : [![Travis develop](https://api.travis-ci.org/sapk/docker-volume-gluster.svg?branch=develop)](https://travis-ci.org/sapk/docker-volume-gluster)


Use GlusterFS as a backend for docker volume

Status : **beta (working)**

Use GlusterFS cli in the plugin container so it depend on fuse on the host.

## Docker plugin (New & Easy method) [![Docker Pulls](https://img.shields.io/docker/pulls/sapk/plugin-gluster.svg)](https://hub.docker.com/r/sapk/plugin-gluster) [![ImageLayers Size](https://img.shields.io/imagelayers/image-size/sapk/plugin-gluster/latest.svg)](https://hub.docker.com/r/sapk/plugin-gluster)
```
docker plugin install sapk/plugin-gluster
docker volume create --driver sapk/plugin-gluster --opt voluri="<volumeserver>:<volumename>" --name test
docker run -v test:/mnt --rm -ti ubuntu
```

## Create and Mount volume
```
docker volume create --driver sapk/plugin-gluster --opt voluri="<volumeserver>,<otherserver>,<otheroptionalserver>:<volumename></optional/sub/dir>" --name test
docker run -v test:/mnt --rm -ti ubuntu
```

## Docker-compose
```
volumes:
  some_vol:
    driver: sapk/plugin-gluster
    driver_opts:
      voluri: "<volumeserver>:<volumename></optional/sub/dir>"
```


## Additionnal docker-plugin config
```
docker plugin disable sapk/plugin-gluster

docker plugin set sapk/plugin-gluster DEBUG=1 #Activate --verbose
docker plugin set sapk/plugin-gluster MOUNT_UNIQ=1 #Activate --mount-uniq

docker plugin enable sapk/plugin-gluster
```

## Install specific version
```
docker plugin install sapk/plugin-gluster:v1.0.11

# Full list of available tags : https://hub.docker.com/r/sapk/plugin-gluster/tags/
```



## Legacy plugin installation
For Docker version 1.12 or below, the managed plugin system is not supported. This also happens if the plugin is not installed via
`docker plugin install`.
[Docker's new plugin system](https://docs.docker.com/engine/extend/) is the preferred way to add drivers and plugins, where the plugin is just
an image downloaded from registry containing the executable and needed configuration files. You can run both legacy and new plugins
in Docker versions above 1.12, but be aware that legacy plugins will not show up on `docker plugin ls`. They will be listed instead under `plugins` on `docker info`.

That way, the driver's name will be just `gluster` (in both the CLI and Compose environments):

#### Build
```
make
```

#### Start daemon
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
  -h, --help         help for daemon
      --mount-uniq   Set mountpoint based on definition and not the name of volume

Global Flags:
  -b, --basedir string   Mounted volume base directory (default "/var/lib/docker-volumes/gluster")
  -v, --verbose          Turns on verbose logging
```

#### Start daemon with systemd

This will enable execution of the daemon on bootup using systemd. On CentOS, this will require `glusterfs-fuse` package to be installed.  

Extract the release file and copy the appropriate binary for your platform into `/usr/bin` and the [docker-volume-gluster.service](https://raw.githubusercontent.com/sapk/docker-volume-gluster/master/support/systemd/docker-volume-gluster.service) into `/etc/systemd/system`.  Then reload the daemon.  Use the following as a guide:

```
install -m 0755 -o root -g root /tmp/build/docker-volume-gluster-linux-amd64 /usr/bin/docker-volume-gluster
install -m 0644 -o root -g root /tmp/build/docker-volume-gluster/docker-volume-gluster.service /etc/systemd/system
systemctl daemon-reload
```

#### Create and Mount volume
```
docker volume create --driver gluster --opt voluri="<volumeserver>:<volumename>" --name test
docker run -v test:/mnt --rm -ti ubuntu
```



## Performances : 
As tested [here](https://github.com/sapk/docker-volume-gluster/issues/10#issuecomment-350126471), this plugin provide same performances as a gluster volume mounted on host via docker bind mount.

## Inspired from :
 - https://github.com/ContainX/docker-volume-netshare/
 - https://github.com/vieux/docker-volume-sshfs/
 - https://github.com/sapk/docker-volume-gvfs
 - https://github.com/calavera/docker-volume-glusterfs
 - https://github.com/codedellemc/rexray
