FROM bitnami/kafka:3.6
USER root
RUN install_packages gettext

ADD ./mm2.template /opt/mirrormaker/mm2.template
ADD ./run.sh /opt/mirrormaker/run.sh
RUN chmod +x /opt/mirrormaker/run.sh

RUN mkdir -p /var/run/mirrormaker
RUN chown 1234 /var/run/mirrormaker

ENV TOPICS .*
ENV DESTINATION "source-cluster:9092"
ENV SOURCE "localhost:9092"
ENV REPLICATION_FACTOR 1
ENV ACLS_ENABLED "false"
ENV TASKS_MAX 10
# REF https://docs.confluent.io/platform/current/connect/references/allconfigs.html#distributed-worker-configuration
ENV GROUP_ID "connect-cluster"
ENV CONFIG_STORAGE_TOPIC "mm2-configs"
ENV OFFSET_STORAGE_TOPIC "mm2-offsets"
ENV STATUS_STORAGE_TOPIC "mm2-status"

EXPOSE 9092 9094

USER 1234
CMD /opt/mirrormaker/run.sh