FROM registry.access.redhat.com/ubi8:8.3
LABEL vendor="Dell Inc." \
      name="csm-resiliency" \
      summary="Dell EMC Container Storage Modules (CSM) for Resiliency" \
      description="Makes Kubernetes applications, including those that utilize persistent storage, more resilient to various failures" \
      version="2.0.0" \
      license="Apache-2.0"
RUN dnf -y install openssl && \
    dnf -y install nettle && \
    dnf -y install gnutls && \
    dnf -y install systemd
COPY podmon /podmon
ENTRYPOINT [ "/podmon" ]
