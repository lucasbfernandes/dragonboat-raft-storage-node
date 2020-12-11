FROM alpine:3.9

USER nobody

ADD build/_output/dragonboat-raft-storage-node /usr/local/bin/dragonboat-raft-storage-node

ENTRYPOINT ["dragonboat-raft-storage-node"]
