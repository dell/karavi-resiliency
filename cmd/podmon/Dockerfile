FROM registry.access.redhat.com/ubi8:8.3
RUN dnf -y install openssl
RUN dnf -y install nettle
RUN dnf -y install gnutls
COPY podmon /podmon
ENTRYPOINT [ "/podmon" ]
