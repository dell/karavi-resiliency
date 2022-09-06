FROM registry.access.redhat.com/ubi8/ubi:8.6-903.1661794351
LABEL vendor="Dell Inc." \
      name="csm-resiliency" \
      summary="Dell Container Storage Modules (CSM) for Resiliency" \
      description="Makes Kubernetes applications, including those that utilize persistent storage, more resilient to various failures" \
      version="2.0.0" \
      license="Apache-2.0"
RUN microdnf -y install openssl && \
    microdnf -y install nettle && \
    microdnf -y install gnutls && \
    microdnf -y install systemd
COPY podmon /podmon
ENTRYPOINT [ "/podmon" ]
