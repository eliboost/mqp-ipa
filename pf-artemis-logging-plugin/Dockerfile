FROM apache/activemq-artemis:2.44.0

COPY ./target/*.jar /opt/activemq-artemis/lib/

COPY ./test/broker.xml /var/lib/artemis-instance/etc-override/