package ch.post.pf.mqpopsui.controller;

import ch.post.pf.mqpopsui.dto.*;
import ch.post.pf.mqpopsui.dto.jolokia.*;
import ch.post.pf.mqpopsui.security.RequireOperatorRole;
import ch.post.pf.mqpopsui.security.UserService;
import ch.post.pf.mqpopsui.services.JmsMessageService;
import ch.post.pf.mqpopsui.services.MessageConverterService;
import ch.post.pf.mqpopsui.services.QueueService;
import ch.post.pf.mqpopsui.util.AmqEndpointResolver;
import ch.post.pf.mqpopsui.util.InstancesListProperties;
import com.fasterxml.jackson.databind.ObjectMapper;
import lombok.extern.slf4j.Slf4j;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.core.ParameterizedTypeReference;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.MultipartFile;
import org.springframework.web.servlet.mvc.method.annotation.StreamingResponseBody;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.URI;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.*;

@RestController
@RequestMapping({"/mqp-ops-ui/api", "/mqp-ops-ui-nonprod/api"})
@Slf4j
public class QueueController extends BaseController {

    static final Logger auditLogger = LoggerFactory.getLogger("ch.post.pf.mqpopsui.audit");
    private final InstancesListProperties instancesListProperties;
    private final UserService userService;
    private final JmsMessageService jmsMessageService;
    private final MessageConverterService messageConverterService;
    private final ObjectMapper objectMapper;
    private final QueueService queueService;
    private final AmqEndpointResolver amqEndpointResolver;

    @Value("${spring.servlet.multipart.location}")
    private String uploadTempDir;

    public QueueController(final InstancesListProperties instancesListProperties, final UserService userService, final JmsMessageService jmsMessageService, final MessageConverterService messageConverterService, final ObjectMapper objectMapper, final QueueService queueService, final AmqEndpointResolver amqEndpointResolver) {
        this.instancesListProperties = instancesListProperties;
        this.userService = userService;
        this.jmsMessageService = jmsMessageService;
        this.messageConverterService = messageConverterService;
        this.objectMapper = objectMapper;
        this.amqEndpointResolver = amqEndpointResolver;
        this.queueService = queueService;
    }

    @RequireOperatorRole
    @GetMapping("/v1/queue/attributes")
    public AMQQueue getQueueAttributes(@RequestParam String instanceName, @RequestParam String queueName, @RequestParam Integer replica, @RequestParam String context) {
        AMQInstance instance = instancesListProperties.getInstanceByNameAndContext(instanceName, context);
        URI amqBrokerUri = amqEndpointResolver.consoleUri(instance, replica);

        // todo: check eror handler, does it work if one of the replicas is offline?
        try {
            JolokiaRequest request = new JolokiaRequest();
            request.setType("READ");
            request.setMbean("org.apache.activemq.artemis:broker=\"" + instance.getBrokerName() + "\",component=addresses,address=\"" + queueName + "\",subcomponent=queues,routing-type=\"anycast\",queue=\"" + queueName + "\"");

            ParameterizedTypeReference<JolokiaAMQQueueAttributesResponse> typeReference = new ParameterizedTypeReference<>() {
            };

            JolokiaAMQQueueAttributesResponse resp = makeRequest(request, amqBrokerUri, typeReference);

            AMQQueue q = resp.getValue();
            q.setReplica(replica);

            return q;

        } catch (Exception e) {
            log.error("Error with an instance {}.", e.getMessage());
            return null;
        }
    }

    @RequireOperatorRole
    @GetMapping("/v1/queue/messagesInQueue")
    public List<AMQMessage> getMessagesInQueue(@RequestParam String instanceName, @RequestParam String queueName, @RequestParam Integer replica, @RequestParam Integer page, @RequestParam Integer pageSize, @RequestParam String context) {
        AMQInstance instance = instancesListProperties.getInstanceByNameAndContext(instanceName, context);
        try {
            return queueService.getAllMessages(instance, queueName, replica, page, pageSize);
        } catch (Exception e) {
            log.error("Error browsing queue '{}' in instance '{}': {}", queueName, instanceName, e.getMessage());
            return null;
        }
    }

    @RequireOperatorRole
    @GetMapping("/v1/queue/searchSingleMessageInQueue")
    public JolokiaAMQMessage searchSingleMessageInQueue(@RequestParam String instanceName, @RequestParam String queueName, @RequestParam Integer replica, @RequestParam String headerMessageId, @RequestParam String context) throws Exception {
        AMQInstance instance = instancesListProperties.getInstanceByNameAndContext(instanceName, context);
        URI amqBrokerUri = amqEndpointResolver.consoleUri(instance, replica);

        JolokiaRequest request = new JolokiaRequest();
        request.setType("EXEC");
        request.setMbean("org.apache.activemq.artemis:broker=\"" + instance.getBrokerName() + "\",component=addresses,address=\"" + queueName + "\",subcomponent=queues,routing-type=\"anycast\",queue=\"" + queueName + "\"");
        request.setOperation("browse(java.lang.String)");
        request.setArguments(new String[]{"Header_messageId='" + headerMessageId + "'"});

        ParameterizedTypeReference<JolokiaAMQMessageResponse> typeReference = new ParameterizedTypeReference<>() {
        };
        JolokiaAMQMessageResponse resp = makeRequest(request, amqBrokerUri, typeReference);

        var value = resp.getValue();
        return (value != null && !value.isEmpty()) ? value.getFirst() : null;
    }

    @RequireOperatorRole
    @PostMapping(value = "/v1/queue/recoverMessage", consumes = MediaType.APPLICATION_JSON_VALUE, produces = MediaType.APPLICATION_JSON_VALUE)
    public boolean recoverSingleMessage(@RequestParam String instanceName, @RequestParam String queueName, @RequestBody String messageId, @RequestParam Integer replica, @RequestParam String context) {
        AMQInstance instance = instancesListProperties.getInstanceByNameAndContext(instanceName, context);
        URI amqBrokerUri = amqEndpointResolver.consoleUri(instance, replica);

        JolokiaRequest request = new JolokiaRequest();
        request.setType("EXEC");
        request.setMbean("org.apache.activemq.artemis:broker=\"" + instance.getBrokerName() + "\",component=addresses,address=\"" + queueName + "\",subcomponent=queues,routing-type=\"anycast\",queue=\"" + queueName + "\"");
        request.setOperation("retryMessage");
        request.setArguments(new String[]{messageId});


        ParameterizedTypeReference<JolokiaStringResponse> typeReference = new ParameterizedTypeReference<>() {
        };
        JolokiaStringResponse resp = makeRequest(request, amqBrokerUri, typeReference);

        return Optional.ofNullable(resp.getValue())
                .map(value -> {
                    auditLogger.info("User {} successfully recovered the message with messageId {} from the queue {} in instance {}.", userService.getUserInfo().get("account"), messageId, queueName, instanceName);
                    return Boolean.parseBoolean(value.getFirst());
                })
                .orElse(false);
    }

    @RequireOperatorRole
    @PostMapping(value = "/v1/queue/changeQueueState", consumes = MediaType.APPLICATION_JSON_VALUE, produces = MediaType.APPLICATION_JSON_VALUE)
    public void changeQueueState(@RequestParam String instanceName, @RequestParam String queueName, @RequestBody ChangeQueueStateRequest operation, @RequestParam Integer replica, @RequestParam String context) throws Exception {
        AMQInstance instance = instancesListProperties.getInstanceByNameAndContext(instanceName, context);
        URI amqBrokerUri = amqEndpointResolver.consoleUri(instance, replica);

        JolokiaRequest request = new JolokiaRequest();
        request.setType("EXEC");
        request.setMbean("org.apache.activemq.artemis:broker=\"" + instance.getBrokerName() + "\",component=addresses,address=\"" + queueName + "\",subcomponent=queues,routing-type=\"anycast\",queue=\"" + queueName + "\"");
        request.setOperation(operation.operation().toJolokiaOperation());

        ParameterizedTypeReference<JolokiaStringResponse> typeReference = new ParameterizedTypeReference<>() {
        };
        makeRequest(request, amqBrokerUri, typeReference);

        auditLogger.info("User {} changed the state of the queue {} in instance {}. Operation: {}", userService.getUserInfo().get("account"), queueName, instanceName, operation);
    }

    @RequireOperatorRole
    @PostMapping(value = "/v1/queue/moveMessages")
    public ResponseEntity<StreamingResponseBody> moveMessages(@RequestBody MoveMessagesRequest request, @RequestParam String instanceName, @RequestParam String context) {
        final String currentQueueName = request.getCurrentQueueName();
        final String newQueueName = request.getNewQueueName();
        final Integer replica = request.getReplica();
        final List<String> messageIds = request.getMessageIds();
        AMQInstance instance = instancesListProperties.getInstanceByNameAndContext(instanceName, context);
        amqEndpointResolver.validate(instance, replica);


        if (messageIds == null || messageIds.isEmpty()) {
            return ResponseEntity.status(HttpStatus.BAD_REQUEST).body(out -> writeNdjson(out, Map.of("type", "error", "message", "No messages provided")));
        }

        final int total = messageIds.size();

        auditLogger.info("User {} initiated moving {} message(s) from {} to {} in instance {}.", userService.getUserInfo().get("account"), messageIds.size(), currentQueueName, newQueueName, instanceName);

        StreamingResponseBody body = out -> {
            try {
                writeNdjson(out, Map.of("type", "progress", "processed", 0, "total", total));
                List<String> failed = jmsMessageService.moveMessages(instance, replica, currentQueueName, newQueueName, messageIds, processed -> {
                    try {
                        writeNdjson(out, Map.of("type", "progress", "processed", processed, "total", total));
                    } catch (IOException e) {
                        throw new RuntimeException(e);
                    }
                });

                messageIds.stream().filter(id -> !failed.contains(id)).forEach(messageId ->
                        auditLogger.info("User {} successfully moved message {} from {} to {} in instance {}.",
                                userService.getUserInfo().get("account"), messageId, currentQueueName, newQueueName, instanceName)
                );
                if (!failed.isEmpty()) {
                    auditLogger.warn("Failed to move {} message(s) from {} to {} in instance {}. Message IDs: {}", failed.size(), currentQueueName, newQueueName, instanceName, failed);
                }

                writeNdjson(out, Map.of("type", "complete", "processed", total - failed.size(), "total", total, "failed", failed.size()));

            } catch (Exception e) {
                auditLogger.error("Error moving messages in instance {}: {}", instanceName, e.getMessage());
                writeNdjson(out, errorEvent(e));
            }
        };

        return ResponseEntity.ok().header("X-Accel-Buffering", "no").contentType(MediaType.parseMediaType("application/x-ndjson")).body(body);
    }

    @RequireOperatorRole
    @PostMapping(value = "/v1/queue/moveAllMessages")
    public ResponseEntity<StreamingResponseBody> moveAllMessages(@RequestBody MoveMessagesRequest request, @RequestParam String instanceName, @RequestParam String context) {
        final String currentQueueName = request.getCurrentQueueName();
        final String newQueueName = request.getNewQueueName();
        final Integer replica = request.getReplica();
        AMQInstance instance = instancesListProperties.getInstanceByNameAndContext(instanceName, context);
        amqEndpointResolver.validate(instance, replica);

        auditLogger.info("User {} initiated moving all message(s) from {} to {} in instance {}.", userService.getUserInfo().get("account"), currentQueueName, newQueueName, instanceName);

        StreamingResponseBody body = out -> {
            writeNdjson(out, Map.of("type", "progress", "processed", 0, "total", -1));

            try {
                int blockSize = request.getBlockSize() != null ? request.getBlockSize() : 1000;
                List<String> failed = jmsMessageService.moveAllMessages(instance, replica, currentQueueName, newQueueName, blockSize, processed -> {
                    try {
                        writeNdjson(out, Map.of("type", "progress", "processed", processed, "total", -1));
                    } catch (Exception e) {
                        throw new RuntimeException(e);
                    }
                });
                if (!failed.isEmpty()) {
                    auditLogger.warn("Failed to move {} message(s) from {} to {} in instance {}. Message IDs: {}", failed.size(), currentQueueName, newQueueName, failed);
                }
                writeNdjson(out, Map.of("type", "complete", "processed", -1, "total", -1, "failed", failed.size()));

            } catch (Exception e) {
                auditLogger.error("Error moving message in instance {}: {}", e.getMessage());
            }

        };


        return ResponseEntity.ok()
                .header("X-Accel-Buffering", "no")
                .contentType(MediaType.parseMediaType("application/x-ndjson"))
                .body(body);
    }

    @RequireOperatorRole
    @PostMapping(value = "/v1/queue/deleteAllMessages")
    public ResponseEntity<StreamingResponseBody> deleteAllMessages(@RequestBody DeleteMessagesRequest request, @RequestParam String instanceName, @RequestParam String context) {
        final String queueName = request.getQueueName();
        final Integer replica = request.getReplica();
        AMQInstance instance = instancesListProperties.getInstanceByNameAndContext(instanceName, context);
        amqEndpointResolver.validate(instance, replica);


        auditLogger.info("User {} initiated deletion of all messages from {} in instance {}", userService.getUserInfo().get("account"), queueName, instanceName);

        StreamingResponseBody body = out -> {
            try {
                int blockSize = request.getBlockSize() != null ? request.getBlockSize() : 1000;
                List<String> failed = jmsMessageService.deleteAllMessages(instance, replica, queueName, blockSize, processed -> {
                    try {
                        writeNdjson(out, Map.of("type", "progress", "processed", processed, "total", -1));
                    } catch (IOException e) {
                        throw new RuntimeException(e);
                    }
                });

                if (!failed.isEmpty()) {
                    auditLogger.warn("Failed to delete {} message(s) from {} in instance {}. Message IDs: {}", failed.size(), queueName, instanceName, failed);
                }
                writeNdjson(out, Map.of("type", "complete", "processed", -1, "total", -1, "failed", failed.size()));
            } catch (Exception e) {
                auditLogger.error("Error deleting all messages in instance {}: {}", instanceName, e.getMessage());
                writeNdjson(out, errorEvent(e));
            }
        };


        return ResponseEntity.ok().header("X-Accel-Buffering", "no")
                .contentType(MediaType.parseMediaType("application/x-ndjson")).body(body);

    }

    @RequireOperatorRole
    @PostMapping(value = "/v1/queue/deleteMessages", produces = "application/x-ndjson")
    public ResponseEntity<StreamingResponseBody> deleteMessages(@RequestBody DeleteMessagesRequest request, @RequestParam String instanceName, @RequestParam String context) {
        final String queueName = request.getQueueName();
        final Integer replica = request.getReplica();
        final List<String> messageIds = request.getMessageIds();
        AMQInstance instance = instancesListProperties.getInstanceByNameAndContext(instanceName, context);
        amqEndpointResolver.validate(instance, replica);

        if (messageIds == null || messageIds.isEmpty()) {
            return ResponseEntity.status(HttpStatus.BAD_REQUEST)
                    .contentType(MediaType.parseMediaType("application/x-ndjson"))
                    .body(out ->
                    writeNdjson(out, Map.of("type", "error", "message", "No messages provided"))
            );
        }
        final int total = messageIds.size();

        auditLogger.info("User {} initiated deletion of {} message(s) from {} in instance {}.", userService.getUserInfo().get("account"), messageIds.size(), queueName, instanceName);

        StreamingResponseBody body = out -> {
            try {
                writeNdjson(out, Map.of("type", "progress", "processed", 0, "total", total));
                List<String> failed = jmsMessageService.deleteMessages(instance, replica, queueName, messageIds, processed -> {
                    try {
                        writeNdjson(out, Map.of("type", "progress", "processed", processed, "total", total));
                    } catch (IOException e) {
                        throw new RuntimeException(e);
                    }
                });
                messageIds.stream().filter(id -> !failed.contains(id)).forEach(messageId ->
                        auditLogger.info("User {} successfully deleted message {} from {} in instance {}.",
                                userService.getUserInfo().get("account"), messageId, queueName, instanceName)
                );
                if (!failed.isEmpty()) {
                    auditLogger.warn("Failed to delete {} message(s) from {} in instance {}. Message IDs: {}", failed.size(), queueName, instanceName, failed);
                }
                writeNdjson(out, Map.of("type", "complete", "processed", total - failed.size(), "total", total, "failed", failed.size()));
            } catch (Exception e) {
                auditLogger.error("Error deleting messages in instance {}: {}", instanceName, e.getMessage());
                writeNdjson(out, errorEvent(e));
            }
        };


        return ResponseEntity.ok()
                .header("X-Accel-Buffering", "no")
                .contentType(MediaType.parseMediaType("application/x-ndjson"))
                .body(body);
    }

    @RequireOperatorRole
    @PostMapping(value = "/v1/queue/sendMessage", consumes = MediaType.APPLICATION_JSON_VALUE, produces = MediaType.APPLICATION_JSON_VALUE)
    public boolean sendSingleMessage(@RequestParam String instanceName, @RequestParam String queueName, @RequestParam String headerMessageId, @RequestParam String headerCommPattern, @RequestParam String headerMandantId, @RequestParam String headerMessageType, @RequestParam String headerPriority, @RequestParam String headerSender, @RequestParam String headerReceiver, @RequestParam String headerSenderPlattform, @RequestParam Integer replica, @RequestBody String messageBody, @RequestParam String context) throws Exception {
        AMQInstance instance = instancesListProperties.getInstanceByNameAndContext(instanceName, context);
        URI amqBrokerUri = amqEndpointResolver.consoleUri(instance, replica);

        JolokiaRequest request = new JolokiaRequest();
        request.setType("EXEC");
        request.setMbean("org.apache.activemq.artemis:broker=\"" + instance.getBrokerName() + "\",component=addresses,address=\"" + queueName + "\",subcomponent=queues,routing-type=\"anycast\",queue=\"" + queueName + "\"");
        request.setOperation("sendMessage(java.util.Map,int,java.lang.String,boolean,java.lang.String,java.lang.String)");
        request.setArguments(new String[]{"{\"Header_messageId\":\"" + headerMessageId + "\", \"Header_commPattern\":\"" + headerCommPattern + "\", \"Header_mandantId\":\"" + headerMandantId + "\", \"Header_messageType\":\"" + headerMessageType + "\", \"Header_priority\":\"" + headerPriority + "\", \"Header_sender\":\"" + headerSender + "\", \"Header_receiver\":\"" + headerReceiver + "\", \"Header_senderPlattform\":\"" + headerSenderPlattform + "\"}", "3", messageBody, "false"});

        ParameterizedTypeReference<JolokiaStringResponse> typeReference = new ParameterizedTypeReference<>() {
        };
        JolokiaStringResponse resp = makeRequest(request, amqBrokerUri, typeReference);

        return Optional.ofNullable(resp.getValue())
                .map(value -> {
                    auditLogger.info("User {} successfully send a message to the queue {} in instance {}. HeaderMessageID: {}", userService.getUserInfo().get("account"), queueName, instanceName, headerMessageId);
                    return Boolean.parseBoolean(value.getFirst());
                })
                .orElseGet(() -> {
                    log.error("Error while sending a message.");
                    return false;
                });
    }

    @RequireOperatorRole
    @PostMapping(value = "/v1/queue/exportMessages", consumes = MediaType.APPLICATION_JSON_VALUE, produces = "application/x-ndjson")
    public ResponseEntity<StreamingResponseBody> exportMessages(@RequestBody ExportMessagesRequest request, @RequestParam String instanceName, @RequestParam String context) {
        AMQInstance instance = instancesListProperties.getInstanceByNameAndContext(instanceName, context);
        amqEndpointResolver.validate(instance, request.getReplica());

        if (request.getMessageIds() == null || request.getMessageIds().isEmpty()) {
            return streamError("No messages provided");
        }

        int total = request.getMessageIds().size();
        String account = userService.getUserInfo().get("account").toString();
        auditLogger.info("User {} initiated export of {} message(s) from {} in instance {}.", account, total, request.getQueueName(), instance.getName());

        StreamingResponseBody body = out -> {
            Path tempFile = createExportTempFile();
            try {
                List<String> failed;
                try (OutputStream xmlOut = Files.newOutputStream(tempFile)) {
                    failed = jmsMessageService.exportMessages(
                            instance, request.getReplica(), request.getQueueName(), request.getMessageIds(),
                            xmlOut,
                            processed -> {
                                try {
                                    writeNdjson(out, Map.of("type", "progress", "processed", processed, "total", total));
                                } catch (IOException e) {
                                    throw new RuntimeException(e);
                                }
                            });
                }

                if (!failed.isEmpty()) {
                    auditLogger.warn("Failed to export {} message(s) from {} in instance {}. Message IDs: {}", failed.size(), request.getQueueName(), instance.getName(), failed);
                    writeNdjson(out, Map.of("type", "error", "message", "Failed to export some messages: " + failed));
                    return;
                }

                streamCompleteNdjson(out, request.getQueueName() + "-export.xml", tempFile);
                auditLogger.info("User {} successfully exported {} message(s) from {} in instance {}.", account, total, request.getQueueName(), instance.getName());

            } catch (Exception e) {
                auditLogger.error("Error exporting messages from {} in instance {}: {}", request.getQueueName(), instance.getName(), e.getMessage());
                writeNdjson(out, errorEvent(e));
            } finally {
                Files.deleteIfExists(tempFile);
            }
        };

        return ResponseEntity.ok().contentType(MediaType.parseMediaType("application/x-ndjson")).body(body);
    }

    @RequireOperatorRole
    @PostMapping(value = "/v1/queue/exportAllMessages", consumes = MediaType.APPLICATION_JSON_VALUE, produces = "application/x-ndjson")
    public ResponseEntity<StreamingResponseBody> exportAllMessages(@RequestBody ExportMessagesRequest request, @RequestParam String instanceName, @RequestParam String context) {
        AMQInstance instance = instancesListProperties.getInstanceByNameAndContext(instanceName, context);
        amqEndpointResolver.validate(instance, request.getReplica());

        int total = request.getTotalCount() != null ? request.getTotalCount() : 0;
        String account = userService.getUserInfo().get("account").toString();
        auditLogger.info("User {} initiated export of all messages from {} in instance {}.", account, request.getQueueName(), instance.getName());

        StreamingResponseBody body = out -> {
            Path tempFile = createExportTempFile();
            try {
                try (OutputStream xmlOut = Files.newOutputStream(tempFile)) {
                    jmsMessageService.exportAllMessages(
                            instance, request.getReplica(), request.getQueueName(), total,
                            xmlOut,
                            processed -> {
                                try {
                                    writeNdjson(out, Map.of("type", "progress", "processed", processed, "total", total));
                                } catch (IOException e) {
                                    throw new RuntimeException(e);
                                }
                            });
                }

                streamCompleteNdjson(out, request.getQueueName() + "-export.xml", tempFile);
                auditLogger.info("User {} successfully exported all messages from {} in instance {}.", account, request.getQueueName(), instance.getName());

            } catch (Exception e) {
                auditLogger.error("Error exporting all messages from {} in instance {}: {}", request.getQueueName(), instance.getName(), e.getMessage());
                writeNdjson(out, errorEvent(e));
            } finally {
                Files.deleteIfExists(tempFile);
            }
        };

        return ResponseEntity.ok().contentType(MediaType.parseMediaType("application/x-ndjson")).body(body);
    }

    private Path createExportTempFile() throws IOException {
        Path dir = Path.of(uploadTempDir);
        Files.createDirectories(dir);
        return Files.createTempFile(dir, "mq-export-", ".xml");
    }

    private void streamCompleteNdjson(OutputStream out, String filename, Path file) throws IOException {
        out.write(("{\"type\":\"complete\",\"filename\":\"" + filename + "\",\"content\":\"").getBytes(StandardCharsets.UTF_8));
        Base64.Encoder encoder = Base64.getEncoder();
        byte[] buf = new byte[3 * 4096];
        try (InputStream fileIn = Files.newInputStream(file)) {
            int read;
            while ((read = fileIn.read(buf)) != -1) {
                out.write(encoder.encode(read == buf.length ? buf : Arrays.copyOf(buf, read)));
            }
        }
        out.write("\"}\n".getBytes(StandardCharsets.UTF_8));
        out.flush();
    }

    private Path createImportFailedMessagesTempFile() throws IOException {
        Path dir = Path.of(uploadTempDir);
        Files.createDirectories(dir);
        return Files.createTempFile(dir, "mq-import-failed-messages-", ".xml");
    }

    private void writeFailedMessagesJournal(Path file, List<ImportFailureArtemis> failures) throws Exception {
        try (OutputStream xmlOut = Files.newOutputStream(file)) {
            var writer = messageConverterService.startJournalStream(xmlOut);
            for (ImportFailureArtemis failure : failures) {
                messageConverterService.writeXmlMessageToStream(writer, failure.message());
            }
            messageConverterService.endJournalStream(writer);
        }
    }

    private void writeFailedWeblogicMessageJournal(Path file, List<ImportFailureWeblogic> failures) throws Exception {
        try (OutputStream xmlOut = Files.newOutputStream(file)) {
            var writer = messageConverterService.startWeblogicStream(xmlOut);
            for (ImportFailureWeblogic failure: failures) {
                messageConverterService.writeWeblogicMessageToStream(writer, failure.message());
            }
        }
    }

    private ResponseEntity<StreamingResponseBody> streamError(String message) {
        StreamingResponseBody body = out -> writeNdjson(out, Map.of("type", "error", "message", message));
        return ResponseEntity.ok().contentType(MediaType.parseMediaType("application/x-ndjson")).body(body);
    }

    private void writeNdjson(OutputStream out, Object data) throws IOException {
        out.write(objectMapper.writeValueAsBytes(data));
        out.write('\n');
        out.flush();
    }

    @RequireOperatorRole
    @PostMapping(value = "/v1/queue/importMessages", consumes = MediaType.MULTIPART_FORM_DATA_VALUE, produces = "application/x-ndjson")
    public ResponseEntity<StreamingResponseBody> importMessages(
            @RequestParam("file") MultipartFile file,
            @RequestParam String instanceName,
            @RequestParam String queueName,
            @RequestParam int replica,
            @RequestParam String context) {

        AMQInstance instance = instancesListProperties.getInstanceByNameAndContext(instanceName, context);
        amqEndpointResolver.validate(instance, replica);

        if (file.isEmpty()) {
            return streamError("Uploaded file is empty");
        }

        if(!queueService.isQueuePartOfInstance(instance.getQueues(), queueName)) return streamError("Provided Queue is not user managed queue");

        auditLogger.info("User {} initiated import to {} in instance {}", userService.getUserInfo().get("account"), queueName, instanceName);
        StreamingResponseBody body = out -> {
            Path tmpFile = Files.createTempFile("mq-import-", ".xml");
            Path failedMessagesFile = null;
            int[] done = {0};
            int[] total = {0};
            int[] failedCount = {0};
            try {
                file.transferTo(tmpFile);
                List<ImportFailureArtemis> failed;
                try (InputStream in = Files.newInputStream(tmpFile)) {
                    failed = jmsMessageService.importMessages(instance, replica, queueName, in, progress -> {
                        done[0] = progress.done();
                        total[0] = progress.total();
                        failedCount[0] = progress.failed();
                        try {
                            writeNdjson(out, Map.of("type", "progress", "done", progress.done(), "total", progress.total(), "failed", progress.failed()));
                        } catch (IOException e) {
                            throw new RuntimeException(e);
                        }
                    });
                }

                if (!failed.isEmpty()) {
                    failedMessagesFile = createImportFailedMessagesTempFile();
                    writeFailedMessagesJournal(failedMessagesFile, failed);
                    auditLogger.warn("Failed to import {} message(s) to {} in instance {}.", failed.size(), queueName, instanceName);

                    out.write(("{\"type\":\"complete\",\"done\":" + done[0] + ",\"total\":" + total[0] + ",\"failed\":" + failedCount[0] +
                            ",\"failedFilename\":\"failed-import-messages.xml\",\"failedContent\":\"").getBytes(StandardCharsets.UTF_8));
                    Base64.Encoder encoder = Base64.getEncoder();
                    byte[] buf = new byte[3 * 4096];
                    try (InputStream fileIn = Files.newInputStream(failedMessagesFile)) {
                        int read;
                        while ((read = fileIn.read(buf)) != -1) {
                            out.write(encoder.encode(read == buf.length ? buf : Arrays.copyOf(buf, read)));
                        }
                    }
                    out.write("\"}\n".getBytes(StandardCharsets.UTF_8));
                    out.flush();
                    return;
                }

                writeNdjson(out, Map.of("type", "complete", "done", done[0], "total", total[0], "failed", failedCount[0]));
                auditLogger.info("User {} successfully imported messages to {} in instance {}", userService.getUserInfo().get("account"), queueName, instanceName);
            } catch (Exception e) {
                auditLogger.error("Error importing messages to {} in instance {}: {}", queueName, instanceName, e.getMessage());
                writeNdjson(out, errorEvent(e));
            } finally {
                Files.deleteIfExists(tmpFile);
                if (failedMessagesFile != null) {
                    Files.deleteIfExists(failedMessagesFile);
                }
            }
        };

        return ResponseEntity.ok().contentType(MediaType.parseMediaType("application/x-ndjson")).body(body);
    }

    @RequireOperatorRole
    @PostMapping(path = "/v1/queue/importWeblogicMessages", consumes = MediaType.MULTIPART_FORM_DATA_VALUE, produces = "application/x-ndjson")
    public ResponseEntity<StreamingResponseBody> importWebLogicMessages(
            @RequestParam("file") MultipartFile file,
            @RequestParam String instanceName,
            @RequestParam String queueName,
            @RequestParam int replica,
            @RequestParam String context
    ) {
        AMQInstance instance = instancesListProperties.getInstanceByNameAndContext(instanceName, context);
        amqEndpointResolver.validate(instance, replica);

        if (file.isEmpty()) {
            return streamError("Uploaded file is empty");
        }

        if(!queueService.isQueuePartOfInstance(instance.getQueues(), queueName)) return streamError("Provided Queue is not user managed queue");

        auditLogger.info("User {} initiated weblogic import to {} in instance {}", userService.getUserInfo().get("account"), queueName, instanceName);
        StreamingResponseBody body = out -> {
            Path tmpFile = Files.createTempFile("mq-import-", ".xml");
            Path failedMessagesFile = null;
            int[] done = {0};
            int[] total = {0};
            int[] failedCount = {0};
            try {
                file.transferTo(tmpFile);
                List<ImportFailureWeblogic> failed;
                try (InputStream in = Files.newInputStream(tmpFile)) {
                    failed = jmsMessageService.importWebLogicMessages(instance, replica, queueName, in, progress -> {
                        done[0] = progress.done();
                        total[0] = progress.total();
                        failedCount[0] = progress.failed();
                        try {
                            writeNdjson(out, Map.of("type", "progress", "done", progress.done(), "total", progress.total(), "failed", progress.failed()));
                        } catch (IOException e) {
                            throw new RuntimeException(e);
                        }
                    });
                }

                if (!failed.isEmpty()) {
                    failedMessagesFile = createImportFailedMessagesTempFile();
                    writeFailedWeblogicMessageJournal(failedMessagesFile, failed);
                    auditLogger.warn("Failed to import {} message(s) to {} in instance {}.", failed.size(), queueName, instanceName);

                    out.write(("{\"type\":\"complete\",\"done\":" + done[0] + ",\"total\":" + total[0] + ",\"failed\":" + failedCount[0] +
                            ",\"failedFilename\":\"failed-import-messages.xml\",\"failedContent\":\"").getBytes(StandardCharsets.UTF_8));
                    Base64.Encoder encoder = Base64.getEncoder();
                    byte[] buf = new byte[3 * 4096];
                    try (InputStream fileIn = Files.newInputStream(failedMessagesFile)) {
                        int read;
                        while ((read = fileIn.read(buf)) != -1) {
                            out.write(encoder.encode(read == buf.length ? buf : Arrays.copyOf(buf, read)));
                        }
                    }
                    out.write("\"}\n".getBytes(StandardCharsets.UTF_8));
                    out.flush();
                    return;
                }

                writeNdjson(out, Map.of("type", "complete", "done", done[0], "total", total[0], "failed", failedCount[0]));
                auditLogger.info("User {} successfully imported messages to {} in instance {}", userService.getUserInfo().get("account"), queueName, instanceName);
            } catch (Exception e) {
                auditLogger.error("Error importing messages to {} in instance {}: {}", queueName, instanceName, e.getMessage());
                writeNdjson(out, errorEvent(e));
            } finally {
                Files.deleteIfExists(tmpFile);
                if (failedMessagesFile != null) {
                    Files.deleteIfExists(failedMessagesFile);
                }
            }
        };


        auditLogger.info("User {} successfully imported weblogic messages to {} in instance {}", userService.getUserInfo().get("account"), queueName, instanceName);
        return ResponseEntity.ok().contentType(MediaType.parseMediaType("application/x-ndjson")).body(body);

    }

    private Map<String, Object> errorEvent(Exception e) {
        String message = e.getMessage();
        if (message == null || message.isBlank()) {
            message = e.getClass().getSimpleName();
        }
        return Map.of("type", "error", "message", message);
    }
}
