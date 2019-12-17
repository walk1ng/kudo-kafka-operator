set -eu

echo "${CRUISE_CONTROL_NAMESPACE},${CRUISE_CONTROL_INSTANCE_NAME},/kafkacruisecontrol/" > /opt/cruise-control/cruise-control-ui/dist/static/config.csv
exec /opt/cruise-control/kafka-cruise-control-start.sh /etc/cruise-control/config/cruise.properties
