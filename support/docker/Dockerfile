FROM debian:testing-slim
MAINTAINER Antoine GIRARD <antoine.girard@sapk.fr>

ENV GOPATH=/usr/share/docker-volume-gluster

ADD . ${GOPATH}

RUN apt-get update && apt-get upgrade -y \
 && apt-get install -y \
          golang glusterfs-client git build-essential \
 && cd $GOPATH && make \
 && mv ./docker-volume-gluster /usr/bin/docker-volume-gluster \
 && cd && mkdir -p /var/lib/docker-volumes/gluster \
 && apt-get remove -y golang git build-essential \
 && apt-get autoremove -y && apt-get autoclean -y && apt-get clean -y \
 && rm -rf $GOPATH /var/lib/apt/lists/*

ENTRYPOINT [ "/usr/bin/docker-volume-gluster" ]
CMD [ "daemon" ]
