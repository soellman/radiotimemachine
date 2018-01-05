FROM scratch

COPY radiotimemachine-linux /radiotimemachine

EXPOSE 8080
ENTRYPOINT ["/radiotimemachine"]
