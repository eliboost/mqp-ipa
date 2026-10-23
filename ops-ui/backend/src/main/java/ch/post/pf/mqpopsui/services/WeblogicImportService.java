package ch.post.pf.mqpopsui.services;


import ch.post.pf.mqpopsui.dto.xml.artemis.XmlMessage;
import ch.post.pf.mqpopsui.dto.xml.artemis.XmlProperty;
import ch.post.pf.mqpopsui.dto.xml.artemis.XmlQueue;
import ch.post.pf.mqpopsui.dto.xml.weblogic.XmlJMSDestination;
import ch.post.pf.mqpopsui.dto.xml.weblogic.XmlWLJMSMessage;
import org.apache.activemq.artemis.api.core.ActiveMQBuffer;
import org.apache.activemq.artemis.api.core.ActiveMQBuffers;
import org.apache.activemq.artemis.api.core.SimpleString;
import org.springframework.stereotype.Service;

import java.time.Instant;
import java.util.ArrayList;
import java.util.Base64;
import java.util.List;

@Service
public class WeblogicImportService {

    public XmlMessage toXmlMessage(XmlWLJMSMessage jmsMessage) {
        XmlMessage xmlMessage = new XmlMessage();

        if (jmsMessage.getHeaders().getMessageId() != null) {
            xmlMessage.setId(Long.parseLong(jmsMessage.getHeaders().getMessageId().split("\\.")[1]));
        } else {
            jmsMessage.getHeaders().getProperties().stream()
                    .filter(p -> "Header_messageId".equals(p.getName()))
                    .map(p -> p.getValue().toString())
                    .findFirst()
                    .orElse("unkown");
        }
        xmlMessage.setPriority(jmsMessage.getHeaders().getPriority() != null ? jmsMessage.getHeaders().getPriority() : 0);
        xmlMessage.setExpiration(jmsMessage.getHeaders().getExpiration() != null ? jmsMessage.getHeaders().getExpiration() : 0);
        xmlMessage.setTimestamp(jmsMessage.getHeaders().getTimestamp() != null ? jmsMessage.getHeaders().getTimestamp() : Instant.now().toEpochMilli());

        if (jmsMessage.getBody().getTextValue() != null) {
            xmlMessage.setType("text");
        } else {
            xmlMessage.setType("bytes");
        }

        List<XmlProperty> properties = new ArrayList<>();

        jmsMessage.getHeaders().getProperties().forEach(xmlProperty -> {
            XmlProperty property = new XmlProperty();
            property.setName(xmlProperty.getName());
            Object val = xmlProperty.getValue();
            property.setValue(val != null ? val.toString() : null);
            property.setType(weblogicValueToArtemisType(val));
            properties.add(property);
        });

        xmlMessage.setProperties(properties);

        XmlJMSDestination jmsDest = jmsMessage.getHeaders().getJmsDestination();
        if (jmsDest != null && jmsDest.getDestination() != null) {
            String destName = jmsDest.getDestination().getName();
            if (destName != null && destName.contains("@")) {
                String queueName = destName.substring(destName.lastIndexOf('@') + 1);
                XmlQueue queue = new XmlQueue();
                queue.setName(queueName);
                xmlMessage.setQueues(List.of(queue));
            }
        }
        if (jmsMessage.getBody().getTextValue() != null) {
            String text = jmsMessage.getBody().getTextValue().toString();
            ActiveMQBuffer buffer = ActiveMQBuffers.dynamicBuffer(text.length() * 2 + 4);
            buffer.writeNullableSimpleString(SimpleString.of(text));
            byte[] bodyBytes = new byte[buffer.readableBytes()];
            buffer.readBytes(bodyBytes);
            xmlMessage.setBody(Base64.getEncoder().encodeToString(bodyBytes));
        } else {
            xmlMessage.setBody(jmsMessage.getBody().getByteValue().toString());
        }

        return xmlMessage;
    }

    private String weblogicValueToArtemisType(Object value) {
        if (value instanceof Long) return "long";
        if (value instanceof Integer) return "integer";
        return "string";
    }
}
