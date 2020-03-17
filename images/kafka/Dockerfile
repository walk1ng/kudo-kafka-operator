FROM golang:alpine AS build-env
WORKDIR /tools
RUN mkdir kafka_utils kafka-connectors-setup
ADD ./kafka-utils ./kafka-utils
ADD ./kafka-connectors-setup ./kafka-connectors-setup
RUN apk add --no-cache gcc musl-dev git mercurial ca-certificates
ENV GO111MODULE=on
RUN cd ./kafka-utils && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o kafka-utils
RUN cd ./kafka-connectors-setup && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o kafka-connectors-setup

FROM debian:stable-20190610 AS java8-builder

RUN apt update -qq && \
    apt install wget -y

RUN wget https://downloads.mesosphere.com/java/openjdk-jre-8u212b03-hotspot-linux-x64.tar.gz -P /srv && \
    mkdir /srv/jdk && \
    tar -xvvf /srv/openjdk-jre-8u212b03-hotspot-linux-x64.tar.gz --strip-components=1 -C /srv/jdk

FROM openjdk:8-jdk as cruise-control
RUN git clone --branch 2.0.77 https://github.com/linkedin/cruise-control.git
WORKDIR cruise-control
RUN ./gradlew jar --warning-mode=all

# using debian:stretch-20190610-slim
FROM debian@sha256:9490c476443a3869e39c2897fa66c91daf5dcbbfca53c976dac7bbdc45775b28

RUN apt-get update && apt-get install -y curl openssl netcat procps
RUN rm -rf /var/lib/apt/lists/*

COPY --from=java8-builder /srv/jdk /usr/share/java

ENV PATH="$PATH:/usr/share/java/bin" \
    JAVA_HOME="/usr/share/java" \
    KAFKA_HOME="/opt/kafka" \
    SCALA_VERSION="2.12" \
    JMX_EXPORTER_PATH="/opt/jmx-exporter" \
    JMX_EXPORTER_VERSION="0.1.0"

ARG KAFKA_VERSION
ARG UNAME=kafka
ARG UID=1000
ARG GID=1000
RUN groupadd -g $GID -o $UNAME && \
    useradd -r -m -u $UID -g $GID $UNAME

RUN curl -O https://archive.apache.org/dist/kafka/${KAFKA_VERSION}/kafka_${SCALA_VERSION}-${KAFKA_VERSION}.tgz && \
    mkdir $KAFKA_HOME && \
    tar xvfz kafka_${SCALA_VERSION}-${KAFKA_VERSION}.tgz -C $KAFKA_HOME --strip-components=1 && \
    rm -f kafka_${SCALA_VERSION}-${KAFKA_VERSION}.tgz*

RUN curl -O "https://repo1.maven.org/maven2/io/prometheus/jmx/jmx_prometheus_javaagent/${JMX_EXPORTER_VERSION}/jmx_prometheus_javaagent-${JMX_EXPORTER_VERSION}.jar" && \
    curl -O "https://repo1.maven.org/maven2/io/prometheus/jmx/jmx_prometheus_javaagent/${JMX_EXPORTER_VERSION}/jmx_prometheus_javaagent-${JMX_EXPORTER_VERSION}.jar.sha1" && \
    echo "$(cat "jmx_prometheus_javaagent-${JMX_EXPORTER_VERSION}.jar.sha1")" "jmx_prometheus_javaagent-${JMX_EXPORTER_VERSION}.jar" | sha1sum -c - && \
    rm "jmx_prometheus_javaagent-${JMX_EXPORTER_VERSION}.jar.sha1" && \
    mkdir -p $JMX_EXPORTER_PATH/config  && \
    mv "jmx_prometheus_javaagent-${JMX_EXPORTER_VERSION}.jar" "/opt/jmx-exporter/jmx_prometheus_javaagent.jar"

RUN curl "https://downloads.mesosphere.com/kafka/assets/kafka-custom-principal-builder-1.0.0.jar" --output ${KAFKA_HOME}/libs/kafka-custom-principal-builder-1.0.0.jar
WORKDIR $KAFKA_HOME

COPY scripts/metrics-config.yml /opt/kafka/
COPY --from=build-env /tools/kafka-utils/ /opt/kafka/
COPY --from=build-env /tools/kafka-connectors-setup/ /opt/kafka/
COPY --from=cruise-control /cruise-control/cruise-control-metrics-reporter/build/libs/cruise-control-metrics-reporter-2.0.77.jar /opt/kafka/libs
RUN chown -R "$UNAME:$UNAME" $KAFKA_HOME

USER $UNAME
