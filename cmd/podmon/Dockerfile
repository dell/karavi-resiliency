FROM registry.access.redhat.com/ubi8:8.3
RUN dnf -y install openssl
COPY podmon /podmon
ENTRYPOINT [ "/podmon" ]
