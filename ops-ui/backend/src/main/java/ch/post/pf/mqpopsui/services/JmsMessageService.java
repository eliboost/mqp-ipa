package ch.post.pf.mqpopsui.services;

import ch.post.pf.mqpopsui.dto.AMQInstance;
import ch.post.pf.mqpopsui.dto.ImportFailureArtemis;
import ch.post.pf.mqpopsui.dto.ImportFailureWeblogic;
import ch.post.pf.mqpopsui.dto.ImportProgress;
import ch.post.pf.mqpopsui.dto.xml.artemis.XmlMessage;
import ch.post.pf.mqpopsui.dto.xml.weblogic.XmlWLJMSMessage;
import ch.post.pf.mqpopsui.jms.JmsConnectionFactory;
import jakarta.jms.Connection;
import jakarta.jms.JMSException;
import jakarta.jms.Session;
import jakarta.xml.bind.JAXBContext;
import jakarta.xml.bind.Unmarshaller;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.apache.activemq.artemis.api.core.ActiveMQException;
import org.apache.activemq.artemis.api.core.SimpleString;
import org.apache.activemq.artemis.api.core.client.ClientConsumer;
import org.apache.activemq.artemis.api.core.client.ClientMessage;
import org.apache.activemq.artemis.api.core.client.ClientProducer;
import org.apache.activemq.artemis.api.core.client.ClientSession;
import org.apache.activemq.artemis.jms.client.ActiveMQConnectionFactory;
import org.apache.activemq.artemis.jms.client.ActiveMQSession;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

import javax.xml.stream.XMLInputFactory;
import javax.xml.stream.XMLStreamReader;
import javax.xml.stream.XMLStreamWriter;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.*;
import java.util.function.Consumer;
import java.util.stream.Collectors;

@Service
@RequiredArgsConstructor
@Slf4j
public class JmsMessageService {

    static final Logger auditLogger = LoggerFactory.getLogger("ch.post.pf.mqpopsui.audit");
    private static final String ORIG_MESSAGE_ID_PROP = "_AMQ_ORIG_MESSAGE_ID";
    private static final String ACTUAL_EXPIRY_PROP = "_AMQ_ACTUAL_EXPIRY";
    private static final String DELIVERY_COUNT_PROP = "_AMQ_DELIVERY_COUNT";

    private final JmsConnectionFactory jmsConnectionFactory;
    private final MessageConverterService messageConverterService;
    private final WeblogicImportService weblogicImportService;
    private final JAXBContext weblogicJaxbContext;

    public List<String> moveAllMessages(AMQInstance instance, int replica, String sourceQueue, String targetQueue, int blockSize, Consumer<Integer> onProgress) throws Exception {
        ActiveMQConnectionFactory factory = jmsConnectionFactory.createConnectionFactory(instance, replica);
        List<String> failed = new ArrayList<>();

        if (blockSize < 1) throw new IllegalArgumentException("blockSize must be > 0");

        try (Connection connection = factory.createConnection()) {
            connection.start();

            try (Session sessionB = connection.createSession(true, Session.SESSION_TRANSACTED)) {
                ClientSession coreSessionB = ((ActiveMQSession) sessionB).getCoreSession();
                ClientProducer targetProducer = coreSessionB.createProducer(targetQueue);
                int messageCount = 0;

                long initialCount = coreSessionB.queueQuery(SimpleString.of(sourceQueue)).getMessageCount();
                long processed = 0;

                try (ClientConsumer consumer = coreSessionB.createConsumer(sourceQueue)) {
                    ClientMessage message;
                    while (processed < initialCount && (message = consumer.receiveImmediate()) != null) {
                        processed++;
                        long internalId = message.getMessageID();
                        try {
                            if (messageCount % blockSize == 0) {
                                sessionB.commit();
                                onProgress.accept(messageCount);
                            }
                            prepareForRequeue(message);
                            targetProducer.send(message);
                            // Required using the Artemis core API (ClientConsumer) bypasses JMS auto-ack. acknowledge() registers the consume within the transaction,
                            // so commit() atomically removes from source AND sends to target.
                            // Without this, commit() only commits the send — source message stays unconsumed.
                            message.acknowledge();
                            log.debug("Moved message {} from {} to {}", internalId, sourceQueue, targetQueue);
                            messageCount++;
                        } catch (Exception e) {
                            failed.add(String.valueOf(internalId));
                            log.debug("Failed to move message {} from {} to {} | reason: {} ", internalId, sourceQueue, targetQueue, e.getMessage());
                            return failed;
                        }
                    }
                }
                sessionB.commit();
                auditLogger.info("Moved all messages from {} to {}", sourceQueue, targetQueue);
            }
        }

        return failed;
    }

    public List<String> moveMessages(AMQInstance instance, int replica, String sourceQueue, String targetQueue, List<String> messageIds, Consumer<Integer> onProgress) throws Exception {
        ActiveMQConnectionFactory factory = jmsConnectionFactory.createConnectionFactory(instance, replica);
        Set<Long> targetIds = messageIds.stream().map(Long::parseLong).collect(Collectors.toSet());
        List<String> failed = new ArrayList<>(messageIds);
        int processed = 0;

        try (Connection connection = factory.createConnection()) {
            connection.start();

            Map<Long, Long> origToInternalId = findUniqueIdentifiers(connection, sourceQueue, targetIds);

            if (origToInternalId == null || origToInternalId.isEmpty()) {
                return failed;
            }

            String filter = origToInternalId.keySet().stream()
                    .map(id -> ORIG_MESSAGE_ID_PROP + " = " + id)
                    .collect(Collectors.joining(" OR "));

            try (Session sessionB = connection.createSession(true, Session.SESSION_TRANSACTED)) {
                ClientSession coreSessionB = ((ActiveMQSession) sessionB).getCoreSession();
                ClientProducer targetProducer = coreSessionB.createProducer(targetQueue);

                try (ClientConsumer consumer = coreSessionB.createConsumer(sourceQueue, filter)) {
                    ClientMessage message;
                    while ((message = consumer.receiveImmediate()) != null) {
                        long origMsgId = message.getLongProperty(ORIG_MESSAGE_ID_PROP);
                        Long internalId = origToInternalId.get(origMsgId);
                        try {
                            if (internalId == null) {
                                log.warn("Skipping unexpected message wi th origId {} in {}", origMsgId, sourceQueue);
                                continue;
                            }
                            prepareForRequeue(message);
                            targetProducer.send(message);
                            message.acknowledge();
                            failed.remove(String.valueOf(internalId));
                            log.debug("Moved message {} (origId: {}) from {} to {}", internalId, origMsgId, sourceQueue, targetQueue);
                        } catch (Exception e) {
                            failed.add(String.valueOf(internalId));
                            log.debug("Failed to move message {} from {} to {} | reason: {}", internalId, sourceQueue, targetQueue, e.getMessage());
                            return failed;
                        }

                        processed++;
                        onProgress.accept(processed);

                    }
                }
                sessionB.commit();
                auditLogger.info("Moved {} messages from {} to {}", messageIds.size() - failed.size(), sourceQueue, targetQueue);
            }
        }

        return failed;
    }




    public List<String> deleteMessages(AMQInstance instance, int replica, String queue, List<String> messageIds) throws Exception {
        return deleteMessages(instance, replica, queue, messageIds, processed -> {
        });
    }

    public List<String> deleteMessages(AMQInstance instance, int replica, String queue, List<String> messageIds, Consumer<Integer> onProgress) throws Exception {
        ActiveMQConnectionFactory factory = jmsConnectionFactory.createConnectionFactory(instance, replica);
        Set<Long> targetIds = messageIds.stream().map(Long::parseLong).collect(Collectors.toSet());
        List<String> failed = new ArrayList<>(messageIds);
        int processed = 0;

        try (Connection connection = factory.createConnection()) {
            connection.start();

            Map<Long, Long> origToInternalId = findUniqueIdentifiers(connection, queue, targetIds);

            if (origToInternalId == null || origToInternalId.isEmpty()) {
                return failed;
            }

            String filter = origToInternalId.keySet().stream()
                    .map(id -> ORIG_MESSAGE_ID_PROP + " = " + id)
                    .collect(Collectors.joining(" OR "));

            try (Session session = connection.createSession(true, Session.SESSION_TRANSACTED)) {
                ClientSession coreSession = ((ActiveMQSession) session).getCoreSession();
                try (ClientConsumer consumer = coreSession.createConsumer(queue, filter)) {
                    ClientMessage message;
                    while ((message = consumer.receiveImmediate()) != null) {
                        long origMsgId = message.getLongProperty(ORIG_MESSAGE_ID_PROP);
                        Long internalId = origToInternalId.get(origMsgId);
                        if (internalId == null) {
                            log.warn("Skipping unexpected message with origId {} in {}", origMsgId, queue);
                            continue;
                        }
                        try {
                            message.acknowledge();
                            failed.remove(String.valueOf(internalId));
                            processed++;
                            onProgress.accept(processed);
                            log.debug("Deleted message {} (origId: {}) from {}", internalId, origMsgId, queue);
                        } catch (Exception e) {
                            failed.add(String.valueOf(internalId));
                            processed++;
                            onProgress.accept(processed);
                            log.debug("Failed to delete message {} (origId: {}) from {} | reason: {} ", internalId, origMsgId, queue, e.getMessage());
                            return failed;
                        }

                    }
                }
                session.commit();
                auditLogger.info("Deleted {} messages from {}", messageIds.size() - failed.size(), queue);
            }
        }

        return failed;
    }


    public List<String> deleteAllMessages(AMQInstance instance, int replica, String queue, int blockSize, Consumer<Integer> onProgress) throws Exception {
        if (blockSize < 1) throw new IllegalArgumentException("blockSize must be > 0");
        ActiveMQConnectionFactory factory = jmsConnectionFactory.createConnectionFactory(instance, replica);
        List<String> failed = new ArrayList<>();

        try (Connection connection = factory.createConnection()) {
            connection.start();

            try (Session session = connection.createSession(true, Session.SESSION_TRANSACTED)) {
                ClientSession coreSession = ((ActiveMQSession) session).getCoreSession();
                try (ClientConsumer consumer = coreSession.createConsumer(queue)) {
                    ClientMessage message;
                    int messageCount = 0;
                    while ((message = consumer.receiveImmediate()) != null) {
                        long internalId = message.getMessageID();
                        try {
                            if (messageCount % blockSize == 0) {
                                session.commit();
                                onProgress.accept(messageCount);
                            }
                            message.acknowledge();
                            messageCount++;
                            log.debug("Deleted message {} from {}", internalId, queue);
                        } catch (Exception e) {
                            failed.add(String.valueOf(internalId));
                            log.error("Failed to delete message {} from {}", internalId, queue, e);
                        }
                    }
                    log.debug("Finished deleting messages. Total deleted: {}, failed: {}", messageCount, failed.size());
                }
                session.commit();
                auditLogger.info("Deleted all messages from {}, failed: {}", queue, failed.size());
            }
        }

        return failed;
    }

    private void prepareForRequeue(ClientMessage message) {
        message.setExpiration(0);
        message.removeProperty(ACTUAL_EXPIRY_PROP);
        message.putIntProperty(DELIVERY_COUNT_PROP, 0);
    }

    public List<String> exportMessages(AMQInstance instance, int replica, String queue, List<String> messageIds, OutputStream xmlOut, Consumer<Integer> onProgress) throws Exception {
        ActiveMQConnectionFactory factory = jmsConnectionFactory.createConnectionFactory(instance, replica);
        Set<Long> targetIds = messageIds.stream().map(Long::parseLong).collect(Collectors.toSet());
        List<String> failed = new ArrayList<>(messageIds);
        int[] processed = {0};

        try (Connection connection = factory.createConnection()) {
            connection.start();

            Map<Long, Long> origToInternalId = findUniqueIdentifiers(connection, queue, targetIds);

            if (origToInternalId == null || origToInternalId.isEmpty()) {
                return failed;
            }

            String filter = origToInternalId.keySet().stream()
                    .map(id -> ORIG_MESSAGE_ID_PROP + " = " + id)
                    .collect(Collectors.joining(" OR "));

            try (Session session = connection.createSession(true, Session.SESSION_TRANSACTED)) {
                ClientSession coreSession = ((ActiveMQSession) session).getCoreSession();
                XMLStreamWriter writer = messageConverterService.startJournalStream(xmlOut);
                try (ClientConsumer consumer = coreSession.createConsumer(queue, filter)) {
                    ClientMessage message;
                    while ((message = consumer.receiveImmediate()) != null) {
                        long origMsgId = message.getLongProperty(ORIG_MESSAGE_ID_PROP);
                        long internalId = origToInternalId.get(origMsgId);
                        messageConverterService.writeMessageToStream(writer, message, queue);
                        failed.remove(String.valueOf(internalId));
                        log.debug("Export message {} (origId: {}) from {}", message.getMessageID(), origMsgId, queue);
                        onProgress.accept(++processed[0]);
                    }
                } finally {
                    messageConverterService.endJournalStream(writer);
                }
                session.rollback();
                auditLogger.info("Exported {} messages from {}", messageIds.size() - failed.size(), queue);
            }
        }

        return failed;
    }

    public void exportAllMessages(AMQInstance instance, int replica, String queue, int maxMessageCount, OutputStream xmlOut, Consumer<Integer> onProgress) throws Exception {
        ActiveMQConnectionFactory factory = jmsConnectionFactory.createConnectionFactory(instance, replica);
        int[] processed = {0};

        try (Connection connection = factory.createConnection()) {
            connection.start();
            try (Session session = connection.createSession(true, Session.SESSION_TRANSACTED)) {
                ClientSession coreSession = ((ActiveMQSession) session).getCoreSession();
                XMLStreamWriter writer = messageConverterService.startJournalStream(xmlOut);
                try (ClientConsumer consumer = coreSession.createConsumer(queue)) {
                    ClientMessage message;
                    while ((message = consumer.receiveImmediate()) != null && processed[0] < maxMessageCount) {
                        messageConverterService.writeMessageToStream(writer, message, queue);
                        log.debug("Export message {} from {}", message.getMessageID(), queue);
                        onProgress.accept(++processed[0]);
                    }
                } finally {
                    messageConverterService.endJournalStream(writer);
                }
                session.rollback();
                auditLogger.info("Exported all {} messages from {}", processed[0], queue);
            }
        }
    }

    public List<ImportFailureArtemis> importMessages(AMQInstance instance, int replica, String queueName, InputStream journalStream, Consumer<ImportProgress> onProgress) throws Exception {
        ActiveMQConnectionFactory factory = jmsConnectionFactory.createConnectionFactory(instance, replica);
        List<ImportFailureArtemis> failed = new ArrayList<>();

        XMLInputFactory xmlFactory = XMLInputFactory.newInstance();
        xmlFactory.setProperty(XMLInputFactory.IS_SUPPORTING_EXTERNAL_ENTITIES, false);
        xmlFactory.setProperty(XMLInputFactory.SUPPORT_DTD, false);
        XMLStreamReader reader = xmlFactory.createXMLStreamReader(journalStream);
        Unmarshaller unmarshaller = messageConverterService.createJournalUnmarshaller();

        try (Connection connection = factory.createConnection()) {
            connection.start();
            try (Session session = connection.createSession(true, Session.SESSION_TRANSACTED)) {
                ClientSession coreSession = ((ActiveMQSession) session).getCoreSession();
                ClientProducer producer = coreSession.createProducer(queueName);

                int index = 0;
                if (reader.hasNext()) reader.next();
                while (reader.getEventType() != XMLStreamReader.END_DOCUMENT) {
                    if (reader.getEventType() == XMLStreamReader.START_ELEMENT && "message".equals(reader.getLocalName())) {
                        XmlMessage xmlMessage = unmarshaller.unmarshal(reader, XmlMessage.class).getValue();
                        try {
                            byte type = messageConverterService.messageTypeFromString(xmlMessage.getType());
                            ClientMessage clientMessage = coreSession.createMessage(type, true);
                            messageConverterService.applyToClientMessage(xmlMessage, clientMessage);
                            producer.send(clientMessage);
                            log.debug("Imported message {} to queue {}", index, queueName);
                        } catch (Exception e) {
                            String failedId = xmlMessage.getId() > 0 ? String.valueOf(xmlMessage.getId()) : String.valueOf(index);
                            failed.add(new ImportFailureArtemis(index, xmlMessage, e.getMessage()));
                            log.error("Failed to import message {} to queue {}: {}", failedId, queueName, e.getMessage());
                        }
                        index++;
                        onProgress.accept(new ImportProgress(index - failed.size(), index, failed.size()));
                    } else {
                        if (reader.hasNext()) reader.next();
                        else break;
                    }
                }
                session.commit();
                auditLogger.info("Imported {} of {} message(s) to {}, failed: {}", index - failed.size(), index, queueName, failed);
            }
        }

        return failed;
    }

    public List<ImportFailureWeblogic> importWebLogicMessages(AMQInstance instance, int replica, String queueName, InputStream weblogicStream, Consumer<ImportProgress> onProgress) throws Exception {
        ActiveMQConnectionFactory factory = jmsConnectionFactory.createConnectionFactory(instance, replica);
        List<ImportFailureWeblogic> failed = new ArrayList<>();

        XMLInputFactory xmlFactory = XMLInputFactory.newInstance();
        xmlFactory.setProperty(XMLInputFactory.IS_SUPPORTING_EXTERNAL_ENTITIES, false);
        xmlFactory.setProperty(XMLInputFactory.SUPPORT_DTD, false);
        XMLStreamReader reader = xmlFactory.createXMLStreamReader(weblogicStream);
        Unmarshaller unmarshaller = weblogicJaxbContext.createUnmarshaller();

        try (Connection connection = factory.createConnection()) {
            connection.start();
            try (Session session = connection.createSession(true, Session.SESSION_TRANSACTED)) {
                ClientSession coreSession = ((ActiveMQSession) session).getCoreSession();
                ClientProducer producer = coreSession.createProducer(queueName);

                int total = 0;
                if (reader.hasNext()) reader.next();
                while (reader.getEventType() != XMLStreamReader.END_DOCUMENT) {
                    if (reader.getEventType() == XMLStreamReader.START_ELEMENT && "WLJMSMessage".equals(reader.getLocalName())) {
                        XmlWLJMSMessage jmsMessage = unmarshaller.unmarshal(reader, XmlWLJMSMessage.class).getValue();
                        total++;
                        String msgId = jmsMessage.getHeaders().getMessageId();
                        if (msgId == null)
                            msgId = jmsMessage.getHeaders().getProperties().stream()
                                    .filter(p -> "Header_messageId".equals(p.getName()))
                                    .map(p -> p.getValue().toString())
                                    .findFirst()
                                    .orElse("unknown");
                        try {
                            XmlMessage xmlMessage = weblogicImportService.toXmlMessage(jmsMessage);
                            byte type = messageConverterService.messageTypeFromString(xmlMessage.getType());
                            ClientMessage clientMessage = coreSession.createMessage(type, true);
                            messageConverterService.applyToClientMessage(xmlMessage, clientMessage);
                            clientMessage.putLongProperty(ORIG_MESSAGE_ID_PROP, UUID.randomUUID().getMostSignificantBits());
                            producer.send(clientMessage);
                            log.debug("Imported weblogic message {} to queue {}", msgId, queueName);
                        } catch (Exception e) {
                            failed.add(new ImportFailureWeblogic(total, jmsMessage, e.getMessage()));
                            log.error("Failed to import weblogic message {} to queue {}: {}", msgId, queueName, e.getMessage());
                        }
                        onProgress.accept(new ImportProgress(total - failed.size(), total, failed.size()));
                    } else {
                        if (reader.hasNext()) reader.next();
                        else break;
                    }
                }
                session.commit();
                auditLogger.info("Imported {} of {} weblogic message(s) to {}, failed: {}", total - failed.size(), total, queueName, failed);
            }
        }

        return failed;
    }

    private Map<Long, Long> findUniqueIdentifiers(Connection connection, String queue, Set<Long> targetIds) throws JMSException, ActiveMQException {
        Map<Long, Long> origToInternalId = new HashMap<>();
        try (Session sessionA = connection.createSession(true, Session.SESSION_TRANSACTED)) {
            ClientSession coreSessionA = ((ActiveMQSession) sessionA).getCoreSession();
            try (ClientConsumer consumer = coreSessionA.createConsumer(queue, true)) {
                ClientMessage message;
                while ((message = consumer.receiveImmediate()) != null) {
                    long internalId = message.getMessageID();
                    if (targetIds.contains(internalId)) {
                        long origMsgId = message.getLongProperty(ORIG_MESSAGE_ID_PROP);
                        origToInternalId.put(origMsgId, internalId);
                    }
                }
            } catch (ActiveMQException e) {
                log.error("Could not find unique identifier for queue {}", queue, e);
                return null;
            }
        }
        return origToInternalId;
    }

}
