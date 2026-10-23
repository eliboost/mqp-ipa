package ch.post.pf.mqpopsui.services;

import ch.post.pf.mqpopsui.dto.xml.artemis.XmlMessage;
import ch.post.pf.mqpopsui.dto.xml.artemis.XmlProperty;
import ch.post.pf.mqpopsui.dto.xml.artemis.XmlQueue;
import ch.post.pf.mqpopsui.dto.xml.weblogic.XmlBody;
import ch.post.pf.mqpopsui.dto.xml.weblogic.XmlDestination;
import ch.post.pf.mqpopsui.dto.xml.weblogic.XmlHeader;
import ch.post.pf.mqpopsui.dto.xml.weblogic.XmlWLJMSMessage;
import jakarta.xml.bind.JAXBContext;
import jakarta.xml.bind.JAXBException;
import jakarta.xml.bind.Unmarshaller;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.apache.activemq.artemis.api.core.ActiveMQBuffer;
import org.apache.activemq.artemis.api.core.Message;
import org.apache.activemq.artemis.api.core.SimpleString;
import org.apache.activemq.artemis.api.core.client.ClientMessage;
import org.springframework.stereotype.Service;

import javax.xml.stream.XMLOutputFactory;
import javax.xml.stream.XMLStreamException;
import javax.xml.stream.XMLStreamWriter;
import java.io.OutputStream;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Base64;
import java.util.List;

@Service
@RequiredArgsConstructor
@Slf4j
public class MessageConverterService {

    private static final String WL_NS = "http://www.bea.com/WLS/JMS/Message";

    private final JAXBContext journalJaxbContext;

    public XmlMessage toXmlMessage(ClientMessage message, String queueName) {
        XmlMessage xml = new XmlMessage();
        xml.setId(message.getMessageID());
        xml.setPriority(message.getPriority());
        xml.setExpiration(message.getExpiration());
        xml.setTimestamp(message.getTimestamp());
        xml.setType(messageTypeToString(message.getType()));

        if (message.getUserID() != null) {
            xml.setUserId(message.getUserID().toString());
        }

        List<XmlProperty> props = new ArrayList<>();
        for (SimpleString propName : message.getPropertyNames()) {
            Object value = message.getObjectProperty(propName);
            XmlProperty prop = new XmlProperty();
            prop.setName(propName.toString());
            prop.setValue(value instanceof byte[] b ? Base64.getEncoder().encodeToString(b) : String.valueOf(value));
            prop.setType(javaTypeToXmlType(value));
            props.add(prop);
        }
        xml.setProperties(props);

        XmlQueue queue = new XmlQueue();
        queue.setName(queueName);
        xml.setQueues(List.of(queue));

        ActiveMQBuffer body = message.getBodyBuffer().duplicate();
        byte[] bytes = new byte[body.readableBytes()];
        body.readBytes(bytes);
        xml.setBody(Base64.getEncoder().encodeToString(bytes));

        return xml;
    }

    public void applyToClientMessage(XmlMessage xmlMessage, ClientMessage target) {
        target.setPriority((byte) xmlMessage.getPriority());
        target.setExpiration(xmlMessage.getExpiration());
        target.setTimestamp(xmlMessage.getTimestamp());

        if (xmlMessage.getProperties() != null) {
            for (XmlProperty prop : xmlMessage.getProperties()) {
                setTypedProperty(target, prop);
            }
        }

        if (xmlMessage.getBody() != null) {
            byte[] bytes = Base64.getDecoder().decode(xmlMessage.getBody().trim());
            target.getBodyBuffer().writeBytes(bytes);
        }
    }

    public XMLStreamWriter startJournalStream(OutputStream out) throws XMLStreamException {
        XMLStreamWriter w = XMLOutputFactory.newInstance().createXMLStreamWriter(out, StandardCharsets.UTF_8.name());
        w.writeStartDocument(StandardCharsets.UTF_8.name(), "1.0");
        w.writeStartElement("activemq-journal");
        w.writeStartElement("messages");
        return w;
    }

    public XMLStreamWriter startWeblogicStream(OutputStream out) throws XMLStreamException {
        XMLStreamWriter w = XMLOutputFactory.newInstance().createXMLStreamWriter(out, StandardCharsets.UTF_8.name());
        w.writeStartDocument(StandardCharsets.UTF_8.name(), "1.0");
        w.writeStartElement("JMSMessageExport");
        return w;
    }

    public void writeMessageToStream(XMLStreamWriter w, ClientMessage message, String queueName) throws XMLStreamException {
        writeXmlMessageElement(w, toXmlMessage(message, queueName));
    }

    public void writeWeblogicMessageToStream(XMLStreamWriter w, XmlWLJMSMessage message) throws XMLStreamException {
        writeWeblogicMessageElement(w, message);
    }

    public void writeXmlMessageToStream(XMLStreamWriter w, XmlMessage message) throws XMLStreamException {
        writeXmlMessageElement(w, message);
    }

    public void endJournalStream(XMLStreamWriter w) throws XMLStreamException {
        w.writeEndElement(); // messages
        w.writeEndElement(); // activemq-journal
        w.writeEndDocument();
        w.flush();
        w.close();
    }

    private void writeWeblogicMessageElement(XMLStreamWriter w, XmlWLJMSMessage msg) throws XMLStreamException {
        String WL_PREFIX = "mes";
        w.writeStartElement(WL_PREFIX, "WLJMSMessage", WL_NS);
        w.writeNamespace(WL_PREFIX, WL_NS);

        XmlHeader header = msg.getHeaders();
        if (header != null) {
            w.writeStartElement(WL_PREFIX, "Header", WL_NS);

            if (header.getMessageId() != null) {
                w.writeStartElement(WL_PREFIX, "JMSMessageID", WL_NS);
                w.writeCharacters(header.getMessageId());
                w.writeEndElement();
            }
            if (header.getDeliveryMode() != null) {
                w.writeStartElement(WL_PREFIX, "JMSDeliveryMode", WL_NS);
                w.writeCharacters(header.getDeliveryMode());
                w.writeEndElement();
            }
            if (header.getJmsDestination() != null && header.getJmsDestination().getDestination() != null) {
                w.writeStartElement(WL_PREFIX, "JMSDestination", WL_NS);
                XmlDestination dest = header.getJmsDestination().getDestination();
                w.writeEmptyElement(WL_PREFIX, "Destination", WL_NS);
                if (dest.getName() != null) w.writeAttribute("name", dest.getName());
                if (dest.getServerName() != null) w.writeAttribute("serverName", dest.getServerName());
                if (dest.getApplicationName() != null) w.writeAttribute("applicationName", dest.getApplicationName());
                if (dest.getModuleName() != null) w.writeAttribute("moduleName", dest.getModuleName());
                w.writeEndElement(); // JMSDestination
            }

            w.writeStartElement(WL_PREFIX, "JMSExpiration", WL_NS);
            w.writeCharacters(String.valueOf(header.getExpiration() != null ? header.getExpiration() : 0L));
            w.writeEndElement();

            w.writeStartElement(WL_PREFIX, "JMSPriority", WL_NS);
            w.writeCharacters(String.valueOf(header.getPriority() != null ? header.getPriority() : 4));
            w.writeEndElement();

            w.writeStartElement(WL_PREFIX, "JMSRedelivered", WL_NS);
            w.writeCharacters(String.valueOf(header.isRedelivered()));
            w.writeEndElement();

            w.writeStartElement(WL_PREFIX, "JMSTimestamp", WL_NS);
            w.writeCharacters(String.valueOf(header.getTimestamp() != null ? header.getTimestamp() : 0L));
            w.writeEndElement();

            if (header.getProperties() != null && !header.getProperties().isEmpty()) {
                w.writeStartElement(WL_PREFIX, "Properties", WL_NS);
                for (ch.post.pf.mqpopsui.dto.xml.weblogic.XmlProperty prop : header.getProperties()) {
                    w.writeStartElement(WL_PREFIX, "property", WL_NS);
                    w.writeAttribute("name", prop.getName());
                    Object val = prop.getValue();
                    if (val instanceof Long) {
                        w.writeStartElement(WL_PREFIX, "Long", WL_NS);
                    } else if (val instanceof Integer) {
                        w.writeStartElement(WL_PREFIX, "Int", WL_NS);
                    } else {
                        w.writeStartElement(WL_PREFIX, "String", WL_NS);
                    }
                    w.writeCharacters(val != null ? String.valueOf(val) : "");
                    w.writeEndElement(); // typed value
                    w.writeEndElement(); // property
                }
                w.writeEndElement(); // Properties
            }

            w.writeEndElement(); // Header
        }

        XmlBody body = msg.getBody();
        if (body != null) {
            w.writeStartElement(WL_PREFIX, "Body", WL_NS);
            if (body.getTextValue() != null) {
                w.writeStartElement(WL_PREFIX, "Text", WL_NS);
                w.writeCharacters(String.valueOf(body.getTextValue()));
                w.writeEndElement();
            } else if (body.getByteValue() != null) {
                w.writeStartElement(WL_PREFIX, "Bytes", WL_NS);
                w.writeCharacters(String.valueOf(body.getByteValue()));
                w.writeEndElement();
            }
            w.writeEndElement(); // Body
        }

        w.writeEndElement(); // WLJMSMessage
    }

    private void writeXmlMessageElement(XMLStreamWriter w, XmlMessage msg) throws XMLStreamException {
        w.writeStartElement("message");
        w.writeAttribute("id", String.valueOf(msg.getId()));
        w.writeAttribute("priority", String.valueOf(msg.getPriority()));
        w.writeAttribute("expiration", String.valueOf(msg.getExpiration()));
        w.writeAttribute("timestamp", String.valueOf(msg.getTimestamp()));
        w.writeAttribute("type", msg.getType());
        if (msg.getUserId() != null) {
            w.writeAttribute("user-id", msg.getUserId());
        }

        w.writeStartElement("properties");
        if (msg.getProperties() != null) {
            for (XmlProperty prop : msg.getProperties()) {
                w.writeEmptyElement("property");
                w.writeAttribute("name", prop.getName());
                w.writeAttribute("value", prop.getValue() != null ? prop.getValue() : "");
                w.writeAttribute("type", prop.getType());
            }
        }
        w.writeEndElement(); // properties

        w.writeStartElement("queues");
        if (msg.getQueues() != null) {
            for (XmlQueue queue : msg.getQueues()) {
                w.writeEmptyElement("queue");
                w.writeAttribute("name", queue.getName());
            }
        }
        w.writeEndElement(); // queues

        w.writeStartElement("body");
        if (msg.getBody() != null) {
            w.writeCData(msg.getBody());
        }
        w.writeEndElement(); // body

        w.writeEndElement(); // message
    }

    public Unmarshaller createJournalUnmarshaller() throws JAXBException {
        return journalJaxbContext.createUnmarshaller();
    }

    public byte messageTypeFromString(String type) {
        return switch (type == null ? "default" : type) {
            case "text" -> Message.TEXT_TYPE;
            case "bytes" -> Message.BYTES_TYPE;
            case "map" -> Message.MAP_TYPE;
            case "stream" -> Message.STREAM_TYPE;
            case "object" -> Message.OBJECT_TYPE;
            default -> Message.DEFAULT_TYPE;
        };
    }

    private void setTypedProperty(ClientMessage message, XmlProperty prop) {
        String name = prop.getName();
        if (name.startsWith("_AMQ_")) {
            log.debug("Ignoring reserved property '{}' from imported message", name);
            return;
        }
        String value = prop.getValue();
        if (value == null) return;
        switch (prop.getType()) {
            case "simple-string" -> message.putStringProperty(SimpleString.of(name), SimpleString.of(value));
            case "string" -> message.putStringProperty(name, value);
            case "long" -> message.putLongProperty(name, Long.parseLong(value));
            case "integer" -> message.putIntProperty(name, Integer.parseInt(value));
            case "byte" -> message.putByteProperty(name, Byte.parseByte(value));
            case "short" -> message.putShortProperty(name, Short.parseShort(value));
            case "double" -> message.putDoubleProperty(name, Double.parseDouble(value));
            case "float" -> message.putFloatProperty(name, Float.parseFloat(value));
            case "boolean" -> message.putBooleanProperty(name, Boolean.parseBoolean(value));
            case "bytes" -> message.putBytesProperty(name, Base64.getDecoder().decode(value));
            default -> message.putStringProperty(name, value);
        }
    }

    private String messageTypeToString(byte type) {
        return switch (type) {
            case Message.TEXT_TYPE -> "text";
            case Message.BYTES_TYPE -> "bytes";
            case Message.MAP_TYPE -> "map";
            case Message.STREAM_TYPE -> "stream";
            case Message.OBJECT_TYPE -> "object";
            default -> "default";
        };
    }

    private String javaTypeToXmlType(Object value) {
        if (value instanceof SimpleString) return "simple-string";
        if (value instanceof String) return "string";
        if (value instanceof Long) return "long";
        if (value instanceof Integer) return "integer";
        if (value instanceof Byte) return "byte";
        if (value instanceof Short) return "short";
        if (value instanceof Double) return "double";
        if (value instanceof Float) return "float";
        if (value instanceof Boolean) return "boolean";
        if (value instanceof byte[]) return "bytes";
        return "string";
    }
}
