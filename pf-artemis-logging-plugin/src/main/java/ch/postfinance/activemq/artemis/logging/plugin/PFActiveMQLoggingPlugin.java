package ch.postfinance.activemq.artemis.logging.plugin;

import org.apache.activemq.artemis.api.core.ActiveMQException;
import org.apache.activemq.artemis.api.core.Message;
import org.apache.activemq.artemis.api.core.SimpleString;
import org.apache.activemq.artemis.core.postoffice.RoutingStatus;
import org.apache.activemq.artemis.core.server.MessageReference;
import org.apache.activemq.artemis.core.server.RoutingContext;
import org.apache.activemq.artemis.core.server.ServerConsumer;
import org.apache.activemq.artemis.core.server.ServerSession;
import org.apache.activemq.artemis.core.server.impl.AckReason;
import org.apache.activemq.artemis.core.server.plugin.ActiveMQServerPlugin;
import org.apache.activemq.artemis.core.transaction.Transaction;
import org.apache.activemq.artemis.core.transaction.TransactionOperationAbstract;
import org.apache.activemq.artemis.core.transaction.impl.XidImpl;
import org.apache.activemq.artemis.spi.core.protocol.RemotingConnection;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.transaction.xa.Xid;
import java.nio.charset.StandardCharsets;
import java.util.*;
import java.util.stream.Collectors;

public class PFActiveMQLoggingPlugin implements ActiveMQServerPlugin {

    private static final Logger LOG = LoggerFactory.getLogger("PFEventLogging");

    public static final String LOG_ALL_EVENTS = "LOG_ALL_EVENTS";
    public static final String LOG_CONNECTION_EVENTS = "LOG_CONNECTION_EVENTS";
    public static final String LOG_SESSION_EVENTS = "LOG_SESSION_EVENTS";
    public static final String LOG_ROUTE_EVENTS = "LOG_ROUTE_EVENTS";
    public static final String LOG_CONSUMER_EVENTS = "LOG_CONSUMER_EVENTS";
    public static final String LOG_DELIVERING_EVENTS = "LOG_DELIVERING_EVENTS";
    public static final String LOG_SENDING_EVENTS = "LOG_SENDING_EVENTS";
    public static final String LOG_XA_EVENTS = "LOG_XA_EVENTS";
    public static final String UNAVAILABLE = "unavailable";

    // Details:
    // must be explicit enabled (still disabled with LOG_ALL_EVENTS)
    public static final String LOG_DETAILS = "LOG_DETAILS";
    // Internal Bridge Store-and-Forward queue events, artemis.internal.sf.my-cluster.xzy
    public static final String LOG_INTERNAL_ADDRESSES = "LOG_INTERNAL_ADDRESSES";

    private boolean logAll = false;
    private boolean logConnectionEvents = false;
    private boolean logSessionEvents = false;
    private boolean logRouteEvents = false;
    private boolean logConsumerEvents = false;
    private boolean logDeliveringEvents = false;
    private boolean logSendingEvents = false;
    private boolean logXAEvents = false;

    private boolean logDetails = false;
    private boolean logInternalAddresses = false;

    private final Map<String, Object> excludeHeaderProperties = new HashMap<>();

    /**
     * used to pass configured properties to Plugin
     */
    @Override
    public void init(Map<String, String> properties) {

        logAll = Boolean.parseBoolean(properties.getOrDefault(LOG_ALL_EVENTS, "false"));
        logConnectionEvents = Boolean.parseBoolean(properties.getOrDefault(LOG_CONNECTION_EVENTS, "false"));
        logSessionEvents = Boolean.parseBoolean(properties.getOrDefault(LOG_SESSION_EVENTS, "false"));
        logRouteEvents = Boolean.parseBoolean(properties.getOrDefault(LOG_ROUTE_EVENTS, "false"));
        logConsumerEvents = Boolean.parseBoolean(properties.getOrDefault(LOG_CONSUMER_EVENTS, "false"));
        logDeliveringEvents = Boolean.parseBoolean(properties.getOrDefault(LOG_DELIVERING_EVENTS, "false"));
        logSendingEvents = Boolean.parseBoolean(properties.getOrDefault(LOG_SENDING_EVENTS, "false"));
        logXAEvents = Boolean.parseBoolean(properties.getOrDefault(LOG_XA_EVENTS, "false"));

        logDetails = Boolean.parseBoolean(properties.getOrDefault(LOG_DETAILS, "false"));
        logInternalAddresses = Boolean.parseBoolean(properties.getOrDefault(LOG_INTERNAL_ADDRESSES, "false"));

        //
        // Populate exclude header properties

        // Don't trace WLS internal JMS header properties.
        // Header properties are passed from WLS queue by OSB flows, not needed
        // Will disappear when all WLS queues are migrated
        excludeHeaderProperties.put("JMS_BEA_DeliveryTime", null);
        excludeHeaderProperties.put("JMS_BEA_RedeliveryLimit", null);
        excludeHeaderProperties.put("JMSXDeliveryCount", null);

        if (!logDetails) {
            // ActiveMQ internal properties, not that interesting
            excludeHeaderProperties.put("__AMQ_CID", null);
            excludeHeaderProperties.put("_AMQ_ROUTING_TYPE", null);
            // Oracle enterprise context id, only usefully for Oracle products (OSB, SOA-Suite)
            excludeHeaderProperties.put("ECIDContext", null);
        }

        dumpConfiguration();
    }

    /**
     * dump the configuration of the logging Plugin
     */
    private void dumpConfiguration() {
        if (LOG.isDebugEnabled()) {
            LOG.debug("LoggingPlugin logAll={}", logAll);
            LOG.debug("LoggingPlugin logConnectionEvents={}", logConnectionEvents);
            LOG.debug("LoggingPlugin logSessionEvents={}", logSessionEvents);
            LOG.debug("LoggingPlugin logRouteEvents={}", logRouteEvents);
            LOG.debug("LoggingPlugin logConsumerEvents={}", logConsumerEvents);
            LOG.debug("LoggingPlugin logSendingEvents={}", logSendingEvents);
            LOG.debug("LoggingPlugin logDeliveringEvents={}", logDeliveringEvents);
            LOG.debug("LoggingPlugin logXAEvents={}", logXAEvents);

            LOG.debug("LoggingPlugin logDetails={}", logDetails);
            LOG.debug("LoggingPlugin logInternalAddresses={}", logInternalAddresses);
        }
    }

    private void logEvent(String text, LogContext logContext) {
        String parametersAsString = logContext.getParametersAsString();
        if (parametersAsString != null) {
            LOG.info("{}: {}", text, parametersAsString);
        } else {
            LOG.info(text);
        }
    }
    private void logEvent(String text, LogContext logContext, Exception e) {
        String parametersAsString = logContext.getParametersAsString();
        if (parametersAsString != null) {
            LOG.info("{}: {}", text, parametersAsString, e);
        } else {
            LOG.info(text, e);
        }
    }
    private void addRemoteConnectionToLogContext(RemotingConnection connection, LogContext logContext) {
        if (connection != null) {
            // Useless: lists all principals for the domain, not explicit the one from client
//            if (logDetails) {
//                if (connection.getSubject() != null && connection.getSubject().getPrincipals() != null) {
//                    int pId = 0;
//                    for (Principal principal : connection.getSubject().getPrincipals()) {
//                        logContext.addParameter("c_principal" + pId, principal.getName());
//                        pId++;
//                    }
//                }
//            }
            logContext.addParameter("ct_connectionId", connection.getID() == null ? UNAVAILABLE : connection.getID().toString());
            logContext.addParameter("ct_clientId", connection.getClientID());
            logContext.addParameter("ct_remoteAddress", connection.getRemoteAddress());
        } else {
            logContext.addParameter("ct_connection", UNAVAILABLE);
        }
    }

    private void addServerSessionToLogContext(ServerSession session, LogContext logContext) {
        if (session != null) {
            logContext.addParameter("s_sessionName", session.getName());
            addRemoteConnectionToLogContext(session.getRemotingConnection(), logContext);
        } else {
            logContext.addParameter("s_session", UNAVAILABLE);
        }
    }

    private void addServerConsumerToLogContext(ServerConsumer consumer, LogContext logContext) {
        if (consumer != null) {
            logContext.addParameter("c_consumerId", consumer.getID());
            logContext.addParameter("c_sessionId", consumer.getSessionID());
            logContext.addParameter("c_queue", consumer.getQueueName() == null ? UNAVAILABLE : consumer.getQueueName().toString());
            logContext.addParameter("c_filter", consumer.getFilterString() == null ? UNAVAILABLE : consumer.getFilterString().toString());
            logContext.addParameter("c_connectionId", consumer.getConnectionID() == null ? UNAVAILABLE : consumer.getConnectionID().toString());
            logContext.addParameter("c_clientId", consumer.getConnectionClientID());
            logContext.addParameter("c_remoteAddress", consumer.getConnectionRemoteAddress());
        } else {
            logContext.addParameter("c_consumer", UNAVAILABLE);
        }
    }

    private void addMessageToLogContext(Message message, LogContext logContext) {
        if (message != null) {
            if (logDetails) {
                logContext.addParameter("m_connectionId", message.getConnectionID()); // is normally null
                logContext.addParameter("m_routingType", message.getRoutingType() == null ? UNAVAILABLE : message.getRoutingType().name());
                logContext.addParameter("m_persistSize", String.valueOf(message.getPersistSize()));
                logContext.addParameter("m_wholeMessageSize", String.valueOf(message.getWholeMessageSize()));
                logContext.addParameter("m_encodeSize", String.valueOf(message.getEncodeSize()));
                logContext.addParameter("m_userId", message.getUserID());
                logContext.addParameter("m_validatedUserId", message.getValidatedUserID());
//                logContext.addParameter("memoryEstimate", String.valueOf(message.getMemoryEstimate()));
                try {
                    logContext.addParameter("m_persistentSize", String.valueOf(message.getPersistentSize()));
                } catch (ActiveMQException e) {
                    logContext.addParameter("m_persistentSize", UNAVAILABLE);
                }
            }
            logContext.addParameter("m_messageId", String.valueOf(message.getMessageID()));
            logContext.addParameter("m_correlationID", message.getCorrelationID() == null ? UNAVAILABLE : message.getCorrelationID().toString());
            logContext.addParameter("m_address", message.getAddress());
            logContext.addParameter("m_size", String.valueOf(message.getEncodeSize()));
            logContext.addParameter("m_isLargeMessage", String.valueOf(message.isLargeMessage()));
            logContext.addParameter("m_isDurable", String.valueOf(message.isDurable()));
            logContext.addParameter("m_isExpired", String.valueOf(message.isExpired()));
            logContext.addParameter("m_priority", String.valueOf(message.getPriority()));
            logContext.addParameter("m_timestamp", String.valueOf(message.getTimestamp()));
            logContext.addParameter("m_scheduledDeliveryTime", String.valueOf(message.getScheduledDeliveryTime()));
            logContext.addParameter("m_expiration", String.valueOf(message.getExpiration()));

            logContext.addParameter("m_timeSinceCreated", String.valueOf(System.currentTimeMillis() - message.getTimestamp()));

            addMessageHeaderToLogContext(message, logContext);

            // Causes NoSuchMethodException: Class org.apache.activemq.artemis.core.message.impl.CoreMessage
            //logContext.addParameter("isPaged", String.valueOf(message.isPaged()));
            //logContext.addParameter("originalEstimate", String.valueOf(message.getOriginalEstimate()));
        } else {
            logContext.addParameter("m_message", UNAVAILABLE);
        }
    }

    private void addMessageHeaderToLogContext(Message message, LogContext logContext) {
        if (message != null) {
            Map<String, Object> headerMap = message.toPropertyMap();
            if (headerMap != null) {
                for (Map.Entry<String, Object> entry : headerMap.entrySet()) {
                    if (!excludeHeaderProperties.containsKey(entry.getKey())) {
                        logContext.addParameter("mh_" + entry.getKey(), entry.getValue());
                    }
                }
            }
        }
    }

    private void addMessageReferenceToLogContext(MessageReference messageRef, LogContext logContext) {
        if (messageRef != null) {
            logContext.addParameter("m_deliveryCount", String.valueOf(messageRef.getDeliveryCount()));
            logContext.addParameter("m_isPaged", String.valueOf(messageRef.isPaged()));
            addMessageToLogContext(messageRef.getMessage(), logContext);
        } else {
            logContext.addParameter("m_messageRef", UNAVAILABLE);
        }
    }

    private void addTransactionToLogContext(Transaction tx, LogContext logContext) {
        if (tx != null) {
            logContext.addParameter("tx_id", Long.toString(tx.getID()));
            logContext.addParameter("tx_state", tx.getState() == null ? UNAVAILABLE : tx.getState().name());
            addXidToLogContext(tx.getXid(), "tx_", logContext);
        } else {
            logContext.addParameter("tx", UNAVAILABLE);
        }
    }

    private void addXidToLogContext(Xid xid, String prefix, LogContext logContext) {
        if (xid != null) {
            //logContext.addParameter(prefix + "xidFormatId", xid.getFormatId()); useless
            if (logDetails) {
                logContext.addParameter(prefix + "xidBranch",
                        xid.getBranchQualifier() == null ? UNAVAILABLE : new String(xid.getBranchQualifier(), StandardCharsets.UTF_8));
            }
            logContext.addParameter(prefix + "xidGlobalIdHex",
                    xid.getGlobalTransactionId() == null ? UNAVAILABLE : xidBytesToHexString(xid.getGlobalTransactionId()));
            logContext.addParameter(prefix + "xidGlobalId",
                    xid.getGlobalTransactionId() == null ? UNAVAILABLE : xidBytesToString(xid.getGlobalTransactionId()));
            logContext.addParameter(prefix + "xidBase64", XidImpl.toBase64String(xid));
        } else {
            logContext.addParameter(prefix + "xid", UNAVAILABLE);
        }
    }

    private String xidBytesToHexString(byte[] bytes) {
        // Convert byte array to a readable hex string
        StringBuilder hexString = new StringBuilder();
        for (byte b : bytes) {
            hexString.append(String.format("%02X", b));
        }
        // Didn't work: new String(xid.getGlobalTransactionId(), StandardCharsets.UTF_8)
        return hexString.toString();
    }

    private String xidBytesToString(byte[] bytes) {
        StringBuilder builder = new StringBuilder();

        for (int i = 0; i < bytes.length; ++i) {
            byte b = bytes[i];
            builder.append(b);
            if (i != bytes.length - 1) {
                builder.append('.');
            }
        }

        return builder.toString();
    }

    protected boolean excludeAddress(MessageReference messageReference) {
        boolean result = false;
        if (messageReference != null) {
            result = excludeAddress(messageReference.getMessage());
        }
        return result;
    }

    protected boolean excludeAddress(Message message) {
        return !logInternalAddresses && isInternalActiveMQAddress(message);
    }

    protected boolean isInternalActiveMQAddress(Message message) {
        boolean result = false;
        if (message != null && message.getAddress() != null) {
            // Internal addresses:
            // activemq.notifications
            // $.artemis.internal.sf.my-cluster.a1403b85-78f4-11f0-b0c3-fa080526517e (plugin is not called for this queue)
            // $sys.mqtt.sessions

            // Default addresses:
            // DLQ
            // ExpiryQueue

            result = message.getAddress().startsWith("activemq");
        }
        return result;
    }

    /**
     * A connection has been created.
     */
    @Override
    public void afterCreateConnection(RemotingConnection connection) {
        if (logAll || logConnectionEvents) {
            LogContext logContext = new LogContext();
            addRemoteConnectionToLogContext(connection, logContext);
            logEvent("Connection created", logContext);
        }
    }

    /**
     * A connection has been destroyed.
     */
    @Override
    public void afterDestroyConnection(RemotingConnection connection) {
        if (logAll || logConnectionEvents) {
            LogContext logContext = new LogContext();
            addRemoteConnectionToLogContext(connection, logContext);
            logEvent("Connection destroyed", logContext);
        }
    }

    /**
     * After a session has been created.
     */
    @Override
    public void afterCreateSession(ServerSession session) {
        if (logAll || logSessionEvents) {
            LogContext logContext = new LogContext();
            addServerSessionToLogContext(session, logContext);
            logEvent("Session created", logContext);
        }
    }

    /**
     * After a session is closed
     */
    @Override
    public void afterCloseSession(ServerSession session, boolean failed) {
        if (logAll || logSessionEvents) {
            LogContext logContext = new LogContext();
            addServerSessionToLogContext(session, logContext);
            logContext.addParameter("failed", String.valueOf(failed));
            logEvent("Session closed", logContext);
        }
    }

    /**
     * After a consumer has been created
     */
    @Override
    public void afterCreateConsumer(ServerConsumer consumer) {
        if (logAll || logConsumerEvents) {
            LogContext logContext = new LogContext();
            addServerConsumerToLogContext(consumer, logContext);
            logEvent("Consumer created", logContext);
        }
    }

    /**
     * After a consumer is closed
     */
    @Override
    public void afterCloseConsumer(ServerConsumer consumer, boolean failed) {
        if (logAll || logConsumerEvents) {
            LogContext logContext = new LogContext();
            addServerConsumerToLogContext(consumer, logContext);
            logContext.addParameter("failed", String.valueOf(failed));
            logEvent("Consumer closed", logContext);
        }
    }

    /**
     * After a message is sent
     */
    @Override
    public void afterSend(ServerSession session,
                          Transaction tx,
                          Message message,
                          boolean direct,
                          boolean noAutoCreateQueue,
                          RoutingStatus result) {
        if ((logAll || logSendingEvents) && !excludeAddress(message)) {
            LogContext logContext = new LogContext();
            addServerSessionToLogContext(session, logContext);
            addMessageToLogContext(message, logContext);
            addTransactionToLogContext(tx, logContext);
            logContext.addParameter("direct", String.valueOf(direct));
            logContext.addParameter("routingStatus", result.name());
            logEvent("Message produced", logContext);

            if (tx != null) {
                tx.addOperation(new TransactionOperationAbstract() {
                    @Override
                    public void afterCommit(Transaction tx) {
                        LogContext logContext = new LogContext();
                        addMessageToLogContext(message, logContext);
                        logEvent("Message produced commit", logContext);
                    }

                    @Override
                    public void afterRollback(Transaction tx) {
                        LogContext logContext = new LogContext();
                        addMessageToLogContext(message, logContext);
                        logEvent("Message produced rollback", logContext);
                    }
                });
            }
        }
    }

    @Override
    public void onSendException(ServerSession session,
                                Transaction tx,
                                Message message,
                                boolean direct,
                                boolean noAutoCreateQueue,
                                Exception e) {
        if ((logAll || logSendingEvents) && !excludeAddress(message)) {
            LogContext logContext = new LogContext();
            addServerSessionToLogContext(session, logContext);
            addMessageToLogContext(message, logContext);
            addTransactionToLogContext(tx, logContext);
            logContext.addParameter("direct", Boolean.toString(direct));
            logEvent("Message produced exception", logContext, e);
        }
    }

    /**
     * After a message is routed
     */
    @Override
    public void afterMessageRoute(Message message,
                                  RoutingContext context,
                                  boolean direct,
                                  boolean rejectDuplicates,
                                  RoutingStatus result) {
        if ((logAll || logRouteEvents) && !excludeAddress(message)) {
            // Exclude internal
            LogContext logContext = new LogContext();
            addMessageToLogContext(message, logContext);
            logContext.addParameter("direct", String.valueOf(direct));
            logContext.addParameter("rejectDuplicates", String.valueOf(rejectDuplicates));
            logContext.addParameter("routingStatus", result.name());
            logEvent("Message routed", logContext);
        }
    }


    @Override
    public void onMessageRouteException(Message message,
                                        RoutingContext context,
                                        boolean direct,
                                        boolean rejectDuplicates,
                                        Exception e) {
        if ((logAll || logRouteEvents) && !excludeAddress(message)) {
            LogContext logContext = new LogContext();
            addMessageToLogContext(message, logContext);
            logContext.addParameter("direct", String.valueOf(direct));
            logEvent("Message route exception", logContext, e);
        }
    }

    /**
     * After a message is delivered to a client consumer
     */
    @Override
    public void afterDeliver(ServerConsumer consumer, MessageReference messageReference) {

        if ((logAll || logDeliveringEvents) && !excludeAddress(messageReference)) {
            LogContext logContext = new LogContext();
            addServerConsumerToLogContext(consumer, logContext);
            addMessageReferenceToLogContext(messageReference, logContext);
            logEvent("Message delivered", logContext);
        }
    }

    @Override
    public void messageExpired(MessageReference messageReference, SimpleString messageExpiryAddress, ServerConsumer consumer) {
        if ((logAll || logDeliveringEvents) && !excludeAddress(messageReference)) {
            LogContext logContext = new LogContext();
            addServerConsumerToLogContext(consumer, logContext);
            addMessageReferenceToLogContext(messageReference, logContext);
            logContext.addParameter("messageExpiryAddress", messageExpiryAddress.toString());
            logEvent("Message expired", logContext);
        }
    }

    /**
     * A message has been acknowledged
     */
    @Override
    public void messageAcknowledged(final Transaction tx, final MessageReference messageReference, final AckReason reason, final ServerConsumer consumer) {
        if ((logAll || logDeliveringEvents) && !excludeAddress(messageReference)) {
            LogContext logContext = new LogContext();
            addServerConsumerToLogContext(consumer, logContext);
            addMessageReferenceToLogContext(messageReference, logContext);
            addTransactionToLogContext(tx, logContext);
            logContext.addParameter("ackReason", reason.name());
            logEvent("Message acknowledged", logContext);

            if (tx != null) {
                tx.addOperation(new TransactionOperationAbstract() {
                    @Override
                    public void afterCommit(Transaction tx) {
                        LogContext logContext = new LogContext();
                        addMessageReferenceToLogContext(messageReference, logContext);
                        logEvent("Message acknowledged commit", logContext);
                    }

                    @Override
                    public void afterRollback(Transaction tx) {
                        LogContext logContext = new LogContext();
                        addMessageReferenceToLogContext(messageReference, logContext);
                        logEvent("Message acknowledged rollback", logContext);
                    }
                });
            }
        }
    }

    @Override
    public void beforePutTransaction(Xid xid, Transaction tx, RemotingConnection connection) throws ActiveMQException {
        if (logAll || logXAEvents) {
            LogContext logContext = new LogContext();
            addXidToLogContext(xid, "xid_", logContext);
            addTransactionToLogContext(tx, logContext);
            addRemoteConnectionToLogContext(connection, logContext);
            logEvent("Transaction add", logContext);
        }
    }

    @Override
    public void beforeRemoveTransaction(Xid xid, RemotingConnection connection) throws ActiveMQException {
        if (logAll || logXAEvents) {
            LogContext logContext = new LogContext();
            addXidToLogContext(xid, "xid_", logContext);
            addRemoteConnectionToLogContext(connection, logContext);
            logEvent("Transaction remove", logContext);
        }
    }
    // Not needed for event logging:
    //   afterPutTransaction()
    //   afterRemoveTransaction()


    static class LogContext {
        record Parameter(String key, String value) {}
        List<Parameter> parameterList = null;

        void addParameter(String key, String value) {
            if (parameterList == null) {
                parameterList = new ArrayList<>();
            }
            parameterList.add(new Parameter(key, value));
        }

        void addParameter(String key, Object value) {
            this.addParameter(key, value.toString());
        }

        void addParameterMap(Map<String, Object> paramMap, String keyPrefix) {
            for (Map.Entry<String, Object> entry : paramMap.entrySet()) {
                addParameter(keyPrefix + entry.getKey(), entry.getValue());
            }
        }

        String getParametersAsString() {
            String result = null;
            if (parameterList != null) {
                result = parameterList.stream()
                        .map( parameter -> parameter.key() + "=" + parameter.value())
                        .collect(Collectors.joining(", "));
            }
            return result;
        }
    }
}

