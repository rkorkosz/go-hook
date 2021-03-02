FROM scratch

COPY go-hook /

EXPOSE 8000
expose 8829/udp

CMD ["/go-hook"]
