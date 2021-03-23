FROM alpine:3.9

USER nobody

ADD build/_output/dragonboat-raft-storage-driver /usr/local/bin/dragonboat-raft-storage-driver

ENTRYPOINT ["dragonboat-raft-storage-driver"]
