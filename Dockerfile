FROM scratch
ADD ca-certificates.crt /etc/ssl/certs/
ADD yuna /
ADD data.json /
CMD ["/yuna"]