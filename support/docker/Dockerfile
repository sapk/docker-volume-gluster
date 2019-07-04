FROM golang AS build-env

COPY . /go/src/github.com/sapk/docker-volume-gluster
WORKDIR /go/src/github.com/sapk/docker-volume-gluster

RUN make build

FROM ubuntu
LABEL maintainer="Antoine GIRARD <antoine.girard@sapk.fr>"

RUN apt-get update \
 && apt-get install -y glusterfs-client \
 && mkdir -p /var/lib/docker-volumes/gluster /etc/docker-volumes/gluster \
 && apt-get autoclean -y && apt-get clean -y \
 && rm -rf /var/lib/apt/lists/*

COPY --from=build-env /go/src/github.com/sapk/docker-volume-gluster/docker-volume-gluster /usr/bin/docker-volume-gluster

RUN /usr/bin/docker-volume-gluster version

ENTRYPOINT [ "/usr/bin/docker-volume-gluster" ]
CMD [ "daemon" ]
