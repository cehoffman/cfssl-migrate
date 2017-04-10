FROM scratch

ADD migrate /

ENTRYPOINT ["/migrate"]
CMD ["-h"]
