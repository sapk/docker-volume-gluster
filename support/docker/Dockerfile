FROM debian:testing-slim
MAINTAINER Antoine GIRARD <antoine.girard@sapk.fr>

ENV GOPATH="/usr/share/docker-volume-gluster"

RUN apt-get install -y
          golang glusterfs-client  \
 && git clone https://github.com/sapk/docker-volume-gluster.git $GOPATH \
 && cd $GOPATH && make \
 && mv ./docker-volume-gluster /usr/bin/docker-volume-gluster \
 && cd && mkdir -p /var/lib/docker-volumes/gluster \
 && apt-get remove -y golang \
 && apt-get autoremove -y && apt-get autoclean -y && apt-get clean -y \
 && rm -rf $GOPATH /var/lib/apt/lists/*

ENTRYPOINT [ "/usr/bin/docker-volume-gluster" ]
CMD [ "daemon" ]
