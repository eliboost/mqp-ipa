package ch.post.pf.mqpopsui.jms;

import ch.post.pf.mqpopsui.dto.AMQInstance;
import ch.post.pf.mqpopsui.util.AmqEndpointResolver;
import lombok.AllArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.apache.activemq.artemis.jms.client.ActiveMQConnectionFactory;
import org.springframework.stereotype.Component;

@Component
@AllArgsConstructor
@Slf4j
public class JmsConnectionFactory {

    private final JmsProperties jmsProperties;
    private final AmqEndpointResolver endpointResolver;

    public ActiveMQConnectionFactory createConnectionFactory(AMQInstance instance, int replica) {
        String host = endpointResolver.acceptorHost(instance, replica);

        String brokerUrl = String.format(
                "(tcp://%s:%d)?sslEnabled=true&verifyHost=false"
                        + "&trustStoreType=JKS&trustStorePath=%s"
                        + "&keyStoreType=PEM&keyStorePath=%s"
                        + "&useTopologyForLoadBalancing=false&ha=false",
                host, jmsProperties.getAcceptorPort(),
                jmsProperties.getTrustStorePath(), jmsProperties.getKeyStorePath());

        return new ActiveMQConnectionFactory(brokerUrl);
    }
}
