FROM alpine:latest

ENV  NAME_CHANGER=/webhooks \
     USER_UID=1001 \
     USER_NAME=name-injector

COPY webhooks /webhooks

ENTRYPOINT ["./webhooks"]

USER ${USER_UID}

