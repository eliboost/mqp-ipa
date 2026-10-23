package ch.postfinance.activemq.artemis.logging.plugin;

import org.apache.activemq.artemis.api.core.Message;
import org.apache.activemq.artemis.core.message.impl.CoreMessage;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static ch.postfinance.activemq.artemis.logging.plugin.PFActiveMQLoggingPlugin.LOG_INTERNAL_ADDRESSES;
import static org.assertj.core.api.Assertions.assertThat;

class PFActiveMQLoggingPluginTest {

    @Test
    void excludeAddress() {
        PFActiveMQLoggingPlugin  plugin = new PFActiveMQLoggingPlugin();
        Map<String, String> properties  = new HashMap<>();
        Message message = new CoreMessage();

        plugin.init(properties);
        message.setAddress("testAddress");
        assertThat(plugin.excludeAddress(message)).isFalse();
        message.setAddress("activemq.notifications");
        assertThat(plugin.excludeAddress(message)).isTrue();
        // No special handling (plugin is not called for this address)
        message.setAddress("$.artemis.internal.sf.my-cluster.a1403b85-78f4-11f0-b0c3-fa080526517e");
        assertThat(plugin.excludeAddress(message)).isFalse();

        properties.clear();
        properties.put(LOG_INTERNAL_ADDRESSES, "true");
        plugin.init(properties);
        message.setAddress("testAddress");
        assertThat(plugin.excludeAddress(message)).isFalse();
        message.setAddress("activemq.notifications");
        assertThat(plugin.excludeAddress(message)).isFalse();
        // No special handling (plugin is not called for this address)
        message.setAddress("artemis.internal.sf.my-cluster.a1403b85-78f4-11f0-b0c3-fa080526517e");
        assertThat(plugin.excludeAddress(message)).isFalse();
    }
}