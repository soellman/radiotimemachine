FROM alpine
RUN apk add -U ca-certificates tzdata

COPY radiotimemachine-linux /radiotimemachine

EXPOSE 8080
ENTRYPOINT ["/radiotimemachine"]
