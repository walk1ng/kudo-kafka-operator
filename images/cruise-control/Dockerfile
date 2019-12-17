FROM openjdk:8-jdk as build-env
ARG CRUISE_CONTROL_VERSION
ARG CRUISE_CONTROL_UI_VERSION
RUN git clone --branch ${CRUISE_CONTROL_VERSION} https://github.com/linkedin/cruise-control.git
WORKDIR cruise-control
RUN ./gradlew jar --warning-mode=all
RUN wget https://github.com/linkedin/cruise-control-ui/releases/download/v${CRUISE_CONTROL_UI_VERSION}/cruise-control-ui-${CRUISE_CONTROL_UI_VERSION}.tar.gz && \
    mkdir /cruise-control-ui && \
    tar -xvvf cruise-control-ui-${CRUISE_CONTROL_UI_VERSION}.tar.gz --strip-components=1 -C /cruise-control-ui && \
    rm cruise-control-ui-${CRUISE_CONTROL_UI_VERSION}.tar.gz
RUN wget https://downloads.mesosphere.com/java/openjdk-jre-8u212b03-hotspot-linux-x64.tar.gz -P /srv && \
    mkdir /srv/jdk && \
    tar -xvvf /srv/openjdk-jre-8u212b03-hotspot-linux-x64.tar.gz --strip-components=1 -C /srv/jdk && \
    rm /srv/openjdk-jre-8u212b03-hotspot-linux-x64.tar.gz

# using debian:stretch-20190610-slim
FROM debian@sha256:9490c476443a3869e39c2897fa66c91daf5dcbbfca53c976dac7bbdc45775b28

RUN apt-get update && apt-get install -y curl openssl netcat procps && \
    rm -rf /var/lib/apt/lists/*

COPY --from=build-env /srv/jdk /usr/share/java
COPY --from=build-env /cruise-control/ /opt/cruise-control/
COPY --from=build-env /cruise-control-ui/ /opt/cruise-control/cruise-control-ui/
COPY scripts/start.sh /opt/cruise-control/
WORKDIR /opt/cruise-control/

ENV PATH="$PATH:/usr/share/java/bin" \
    JAVA_HOME="/usr/share/java"

ARG CRUISE_HOME=/opt/cruise-control
ARG UNAME=cruise
ARG UID=1000
ARG GID=1000

RUN groupadd -g $GID -o $UNAME && \
    useradd -r -m -u $UID -g $GID $UNAME
RUN chown -R "$UNAME:$UNAME" $CRUISE_HOME
