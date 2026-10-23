package ch.post.pf.mqpopsui.services;

import ch.post.pf.mqpopsui.dto.AMQInstance;
import ch.post.pf.mqpopsui.jms.JmsConnectionFactory;
import jakarta.jms.Connection;
import jakarta.jms.Session;
import org.apache.activemq.artemis.api.core.client.ClientConsumer;
import org.apache.activemq.artemis.api.core.client.ClientMessage;
import org.apache.activemq.artemis.api.core.client.ClientProducer;
import org.apache.activemq.artemis.api.core.client.ClientSession;
import org.apache.activemq.artemis.jms.client.ActiveMQConnectionFactory;
import org.apache.activemq.artemis.jms.client.ActiveMQSession;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

/**
 *
 * The service has a two-phase pattern per operation:
 * Phase 1 (findUniqueIdentifiers): sessionA browses the queue without a filter to map
 * internalId (what the UI knows) → origMsgId (_AMQ_ORIG_MESSAGE_ID property).
 * Phase 2 (move/delete): sessionB opens a filtered consumer using the origMsgIds from phase 1,
 * then sends (move) or only acknowledges (delete) each message.
 *
 */
@ExtendWith(MockitoExtension.class)
class JmsMessageServiceTest {

    private static final String SOURCE_QUEUE = "SOURCE.QUEUE";
    private static final String TARGET_QUEUE = "TARGET.QUEUE";
    private static final String ORIG_MSG_ID_PROP = "_AMQ_ORIG_MESSAGE_ID";

    // IDs as seen by the UI (internal Artemis message IDs)
    private static final long INTERNAL_ID_1 = 100L;
    private static final long INTERNAL_ID_2 = 200L;

    // Artemis _AMQ_ORIG_MESSAGE_ID values (set on re-delivered messages)
    private static final long ORIG_ID_1 = 1001L;
    private static final long ORIG_ID_2 = 2002L;

    @Mock
    private JmsConnectionFactory jmsConnectionFactory;

    @InjectMocks
    private JmsMessageService jmsMessageService;

    @Mock
    private ActiveMQConnectionFactory connectionFactory;
    @Mock
    private Connection connection;

    // Phase 1 – browse session (findUniqueIdentifiers)
    @Mock
    private ActiveMQSession sessionA;
    @Mock
    private ClientSession coreSessionA;
    @Mock
    private ClientConsumer browseConsumer;

    // Phase 2 – filtered operation session (move / delete)
    @Mock
    private ActiveMQSession sessionB;
    @Mock
    private ClientSession coreSessionB;
    @Mock
    private ClientConsumer filteredConsumer;
    @Mock
    private ClientProducer targetProducer;

    // Messages returned by the consumers
    @Mock
    private ClientMessage message1;
    @Mock
    private ClientMessage message2;

    private AMQInstance instance;

    @BeforeEach
    void setUp() throws Exception {
        instance = new AMQInstance();
        instance.setName("e1-mqp-test-a");
        instance.setNamespace("test-ns");
        instance.setContext("test-ctx");

        when(jmsConnectionFactory.createConnectionFactory(any(), anyInt())).thenReturn(connectionFactory);
        when(connectionFactory.createConnection()).thenReturn(connection);

        // First createSession call → phase 1 (browse), second → phase 2 (operation)
        when(connection.createSession(true, Session.SESSION_TRANSACTED)).thenReturn(sessionA, sessionB);

        when(sessionA.getCoreSession()).thenReturn(coreSessionA);
        // lenient: Mockito's inline mock maker does not track stub-usage for concrete class
        // methods reliably, so this generates a false UnnecessaryStubbingException otherwise.
        lenient().when(sessionB.getCoreSession()).thenReturn(coreSessionB);

        when(coreSessionA.createConsumer(anyString(), anyBoolean())).thenReturn(browseConsumer);
    }

    /**
     * Stubs both messages so the browse phase (phase 1) maps their internalId → origMsgId.
     * Call this in every test where messages are expected to be found in the queue.
     */
    private void givenBothMessagesInQueue() throws Exception {
        when(message1.getMessageID()).thenReturn(INTERNAL_ID_1);
        when(message1.getLongProperty(ORIG_MSG_ID_PROP)).thenReturn(ORIG_ID_1);
        when(message2.getMessageID()).thenReturn(INTERNAL_ID_2);
        when(message2.getLongProperty(ORIG_MSG_ID_PROP)).thenReturn(ORIG_ID_2);
        when(browseConsumer.receiveImmediate()).thenReturn(message1, message2, null);
    }

    // -------------------------------------------------------------------------
    // moveMessages
    // -------------------------------------------------------------------------

    @Test
    void moveMessages_allMessagesFound_returnsEmptyFailedList() throws Exception {
        givenBothMessagesInQueue();
        when(coreSessionB.createProducer(TARGET_QUEUE)).thenReturn(targetProducer);
        when(coreSessionB.createConsumer(eq(SOURCE_QUEUE), anyString())).thenReturn(filteredConsumer);
        when(filteredConsumer.receiveImmediate()).thenReturn(message1, message2, null);

        List<String> failed = jmsMessageService.moveMessages(
                instance, 0, SOURCE_QUEUE, TARGET_QUEUE, List.of("100", "200"), onProgress -> {});

        assertThat(failed).isEmpty();
        verify(targetProducer).send(message1);
        verify(targetProducer).send(message2);
        verify(message1).acknowledge();
        verify(message2).acknowledge();
        verify(sessionB).commit();
    }

    @Test
    void moveMessages_messagesNotInQueue_returnsAllIdsAsFailed() throws Exception {
        // Browse finds nothing → origToInternalId is empty → early return, phase 2 never runs
        when(browseConsumer.receiveImmediate()).thenReturn(null);

        List<String> failed = jmsMessageService.moveMessages(
                instance, 0, SOURCE_QUEUE, TARGET_QUEUE, List.of("100", "200"), onProgress -> {});

        assertThat(failed).containsExactlyInAnyOrder("100", "200");
        verify(coreSessionB, never()).createProducer(anyString());
        verify(sessionB, never()).commit();
    }

    @Test
    void moveMessages_onlyOneMessageFoundInQueue_returnsOtherIdAsFailed() throws Exception {
        // Browse finds message1 (ID 100) but not message2 (ID 200)
        when(message1.getMessageID()).thenReturn(INTERNAL_ID_1);
        when(message1.getLongProperty(ORIG_MSG_ID_PROP)).thenReturn(ORIG_ID_1);
        ClientMessage irrelevantMessage = mock(ClientMessage.class);
        when(irrelevantMessage.getMessageID()).thenReturn(999L); // not in the requested set
        when(browseConsumer.receiveImmediate()).thenReturn(message1, irrelevantMessage, null);

        when(coreSessionB.createProducer(TARGET_QUEUE)).thenReturn(targetProducer);
        when(coreSessionB.createConsumer(eq(SOURCE_QUEUE), anyString())).thenReturn(filteredConsumer);
        when(filteredConsumer.receiveImmediate()).thenReturn(message1, null);

        List<String> failed = jmsMessageService.moveMessages(
                instance, 0, SOURCE_QUEUE, TARGET_QUEUE, List.of("100", "200"), onProgress -> {});

        assertThat(failed).containsExactly("200");
        verify(targetProducer).send(message1);
        verify(targetProducer, never()).send(message2);
        verify(sessionB).commit();
    }

    // -------------------------------------------------------------------------
    // deleteMessages
    // -------------------------------------------------------------------------

    @Test
    void deleteMessages_allMessagesFound_returnsEmptyFailedList() throws Exception {
        givenBothMessagesInQueue();
        when(coreSessionB.createConsumer(eq(SOURCE_QUEUE), anyString())).thenReturn(filteredConsumer);
        when(filteredConsumer.receiveImmediate()).thenReturn(message1, message2, null);

        List<String> failed = jmsMessageService.deleteMessages(
                instance, 0, SOURCE_QUEUE, List.of("100", "200"));

        assertThat(failed).isEmpty();
        verify(message1).acknowledge();
        verify(message2).acknowledge();
        verify(sessionB).commit();
        // Delete must never forward messages to another queue
        verify(coreSessionB, never()).createProducer(anyString());
    }

    @Test
    void deleteMessages_messagesNotInQueue_returnsAllIdsAsFailed() throws Exception {
        when(browseConsumer.receiveImmediate()).thenReturn(null);

        List<String> failed = jmsMessageService.deleteMessages(
                instance, 0, SOURCE_QUEUE, List.of("100", "200"));

        assertThat(failed).containsExactlyInAnyOrder("100", "200");
        verify(sessionB, never()).commit();
    }

    @Test
    void deleteMessages_onlyOneMessageFoundInQueue_returnsOtherIdAsFailed() throws Exception {
        when(message1.getMessageID()).thenReturn(INTERNAL_ID_1);
        when(message1.getLongProperty(ORIG_MSG_ID_PROP)).thenReturn(ORIG_ID_1);
        ClientMessage irrelevantMessage = mock(ClientMessage.class);
        when(irrelevantMessage.getMessageID()).thenReturn(999L);
        when(browseConsumer.receiveImmediate()).thenReturn(message1, irrelevantMessage, null);

        when(coreSessionB.createConsumer(eq(SOURCE_QUEUE), anyString())).thenReturn(filteredConsumer);
        when(filteredConsumer.receiveImmediate()).thenReturn(message1, null);

        List<String> failed = jmsMessageService.deleteMessages(
                instance, 0, SOURCE_QUEUE, List.of("100", "200"));

        assertThat(failed).containsExactly("200");
        verify(message1).acknowledge();
        verify(message2, never()).acknowledge();
        verify(sessionB).commit();
    }
}
