# Limit Message Size

ActiveMQ does not natively enforce message size limits.  
To prevent oversized messages (e.g. >50MB) from being sent and potentially overloading consumers or the broker, a custom 
validation can be required.

The following implementation that calculates the size of the outgoing message (XML-serialized) blocks transmission if the size 
exceeds 50MB.

## Message Validation

**File:** [`LayoutDocumentRequestSender.java`](https://gitlab.pnet.ch/apmpf/dirm/dirm-rm-core/-/blob/release/2508.x/dirm-rm-services/src/main/java/ch/postfinance/dirm/service/umsysteme/esbpf/deva/LayoutDocumentRequestSender.java)

```java
public static final String DIRM_0113_LARGEMESSAGES_ERROR = "DIRM-0113";
private static final int MAX_MESSAGE_SIZE_BYTES = 50 * 1024 * 1024; // 50 MB

public void sendMessage(final MetaData message, final String orderId, final EmpfaengerEnum empfaenger, final int versandIndex) {
    try {
        int messageSize = calculateMessageSizeInBytes(message);

        if (messageSize > MAX_MESSAGE_SIZE_BYTES) {
            log.error("{}: DIRM-DEVA message exceeds 50MB limit ({} bytes) – OrderId: {}, Empfaenger: {}, Index: {}",
                DIRM_0113_LARGEMESSAGES_ERROR, messageSize, orderId, empfaenger, versandIndex);
            return; // 🛑 Message will not be sent
        }
    } catch (Exception e) {
        log.error("Error calculating message size – OrderId: {}", orderId, e);
        return; // 🛑 Block sending on error
    }

    // ✅ Only send message if size is valid
    this.messageQueueSender.sendAsyncMessageToOutQueue(
        message,
        this.createEsbMetadataAsyncParameters(orderId,
            message.getOutbound().getCustomerLayout().getMetaData().getHeader(),
            empfaenger, versandIndex));
}
```

### Size Calculation Utility

```java
int calculateMessageSizeInBytes(MetaData message) throws JAXBException {
    StringWriter writer = new StringWriter();
    Marshaller marshaller = jaxbContext.createMarshaller();
    marshaller.marshal(message, writer);
    return writer.toString().getBytes(StandardCharsets.UTF_8).length;
}
```
