FROM alpine
RUN apk add -U tzdata

COPY radiotimemachine-linux /radiotimemachine

EXPOSE 8080
ENTRYPOINT ["/radiotimemachine"]
