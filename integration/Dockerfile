# Integration harness: build the yup-find binary for Linux, then run it inside a
# Debian image (GNU coreutils) so its flags, inputs, and outputs can be checked
# against the real `find` reference. Invoked by `make integration`.
FROM golang:1.26 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/yup-find .

FROM debian:stable-slim
RUN apt-get update && apt-get install -y --no-install-recommends findutils && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/yup-find /usr/local/bin/yup-find
COPY integration/run.sh /run.sh
ENTRYPOINT ["/bin/sh", "/run.sh"]
