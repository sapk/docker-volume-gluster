FROM golang AS build-env

COPY . /go/src/app
WORKDIR /go/src/app

RUN make build

FROM ubuntu
MAINTAINER Antoine GIRARD <antoine.girard@sapk.fr>

RUN apt-get update \
 && apt-get install -y glusterfs-client \
 && mkdir -p /var/lib/docker-volumes/gluster /etc/docker-volumes/gluster \
 && apt-get autoclean -y && apt-get clean -y \
 && rm -rf /var/lib/apt/lists/*

COPY --from=build-env /go/src/app/docker-volume-gluster /usr/bin/docker-volume-gluster

RUN /usr/bin/docker-volume-gluster version

ENTRYPOINT [ "/usr/bin/docker-volume-gluster" ]
CMD [ "daemon" ]
