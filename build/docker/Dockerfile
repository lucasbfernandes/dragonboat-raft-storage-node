FROM alpine:3.9

USER nobody

ADD build/_output/dragonboat-raft-replica /usr/local/bin/dragonboat-raft-replica

ENTRYPOINT ["dragonboat-raft-replica"]
