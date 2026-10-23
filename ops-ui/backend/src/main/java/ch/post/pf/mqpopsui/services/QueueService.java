package ch.post.pf.mqpopsui.services;

import ch.post.pf.mqpopsui.dto.AMQInstance;
import ch.post.pf.mqpopsui.dto.AMQMessage;
import ch.post.pf.mqpopsui.dto.jolokia.JolokiaAMQMessageResponse;
import ch.post.pf.mqpopsui.dto.jolokia.JolokiaRequest;
import ch.post.pf.mqpopsui.mapper.MessageMapper;
import ch.post.pf.mqpopsui.util.AmqEndpointResolver;
import lombok.AllArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.core.ParameterizedTypeReference;
import org.springframework.stereotype.Service;

import java.net.URI;
import java.util.List;

@Service
@AllArgsConstructor
@Slf4j
public class QueueService extends BaseService {

    private final MessageMapper messageMapper;
    private final AmqEndpointResolver amqEndpointResolver;

    public List<AMQMessage> getAllMessages(AMQInstance instance, String queueName, Integer replica, Integer page, Integer pageSize) {
        URI amqBrokerUri = amqEndpointResolver.consoleUri(instance, replica);

        JolokiaRequest request = new JolokiaRequest();
        request.setType("EXEC");
        request.setMbean("org.apache.activemq.artemis:broker=\"" + instance.getBrokerName() + "\",component=addresses,address=\"" + queueName + "\",subcomponent=queues,routing-type=\"anycast\",queue=\"" + queueName + "\"");
        request.setOperation("browse(int, int)");
        request.setArguments(new String[]{page.toString(), pageSize.toString()});

        try {
            ParameterizedTypeReference<JolokiaAMQMessageResponse> typeReference = new ParameterizedTypeReference<>() {};
            JolokiaAMQMessageResponse resp = makeRequest(request, amqBrokerUri, typeReference);

            if (!resp.isSuccess()) {
                log.error("Jolokia denied browse on queue '{}' in instance '{}': status={}, error={}", queueName, instance.getName(), resp.getStatus(), resp.getError());
                return null;
            }

            return messageMapper.toAMQMessageList(resp.getValue());
        } catch (Exception ignored) {
            return null;
        }

    }

    public boolean isQueuePartOfInstance(List<String> instanceQueues, String queueName) {
        String cleanName = queueName.replaceAll("(_DLQ|_EXPIRY)$", "");
        return instanceQueues.contains(cleanName);
    }
}
