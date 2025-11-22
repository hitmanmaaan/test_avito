FROM ubuntu:latest
LABEL authors="egorlevchuk"

ENTRYPOINT ["top", "-b"]