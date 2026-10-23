# Custom PF Logging Plugin for ActiveMQ Artemis

This project holds a [plugin](https://activemq.apache.org/components/artemis/documentation/latest/broker-plugins.html) 
for the ActiveMQ Artemis Broker that enhances the default logging with more detailed logs of the message creation and 
consumption. 

## Usage

To use the plugin, the plugin jar needs to be copied to `${CONFIG_INSTANCE_DIR}/lib/`. In the broker.xml the 
plugin can be activated as described below:

```xml
<broker-plugins>
    <broker-plugin class-name="ch.postfinance.activemq.artemis.logging.plugin.PFActiveMQLoggingPlugin">
        <property key="LOG_ALL_EVENTS" value="false"/>
        <property key="LOG_DETAILS" value="false"/>
        <property key="LOG_CONNECTION_EVENTS" value="false"/>
        <property key="LOG_SESSION_EVENTS" value="false"/>
        <property key="LOG_ROUTE_EVENTS" value="false"/>
        <property key="LOG_CONSUMER_EVENTS" value="true"/>
        <property key="LOG_DELIVERING_EVENTS" value="true"/>
        <property key="LOG_SENDING_EVENTS" value="true"/>
        <property key="LOG_INTERNAL_EVENTS" value="false"/>
        <property key="LOG_XA_EVENTS" value="false"/>
    </broker-plugin>
</broker-plugins>
```
## Test

We use terratest to bootstrap an artemis broker with docker-compose. Afterward we do some http 
calls to check if the plugin is in place, activated and works as desired.
The tests can be found here: test/plugin_test.go

