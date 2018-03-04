# Build stage
FROM golang:alpine AS build-env
LABEL Maintainer="Vladislav Mostovoi <vladkimo@gmail.com>" \
      Description="Docker image for Rancher service upgrade tool based on Alpine Linux."

ENV APP_NAME=rancher-service-up

ADD . /usr/local/go/src/$APP_NAME

RUN apk update \
    && apk upgrade \
    && apk --no-cache add make \    
    && cd /usr/local/go/src/$APP_NAME && make build

# Final stage
FROM alpine

ENV APP_NAME=rancher-service-up

COPY --from=build-env /usr/local/go/src/${APP_NAME}/build/${APP_NAME} /usr/local/bin

RUN chmod 0775 /usr/local/bin/${APP_NAME}

CMD ${APP_NAME}
