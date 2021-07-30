FROM registry.access.redhat.com/ubi8:8.3
RUN dnf -y install openssl && \
    dnf -y install nettle && \
    dnf -y install gnutls && \
    dnf -y install systemd
COPY podmon /podmon
ENTRYPOINT [ "/podmon" ]
