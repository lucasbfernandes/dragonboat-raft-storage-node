FROM alpine:3.9

USER nobody

ADD build/_output/dragonboat-raft-storage-proxy /usr/local/bin/dragonboat-raft-storage-proxy

ENTRYPOINT ["dragonboat-raft-storage-proxy"]
