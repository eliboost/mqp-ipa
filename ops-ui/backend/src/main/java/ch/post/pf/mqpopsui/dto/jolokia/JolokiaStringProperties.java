package ch.post.pf.mqpopsui.dto.jolokia;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.ToString;

@ToString
@JsonIgnoreProperties(ignoreUnknown = true)
public class JolokiaStringProperties {

    private String amqOrigQueue;
    private String amqOrigAddress;
    private String headerCommPattern;
    private String headerMandantId;
    private String headerMessageId;
    private String headerMessageType;
    private String headerPriority;
    private String headerReceiver;
    private String headerSender;
    private String headerSenderPlattform;

    @JsonProperty("amqOrigQueue")
    public String getAmqOrigQueue() {
        return amqOrigQueue;
    }

    @JsonProperty("_AMQ_ORIG_QUEUE")
    public void setAmqOrigQueue(String amqOrigQueue) {
        this.amqOrigQueue = amqOrigQueue;
    }

    @JsonProperty("amqOrigAddress")
    public String getAmqOrigAddress() {
        return amqOrigAddress;
    }

    @JsonProperty("_AMQ_ORIG_ADDRESS")
    public void setAmqOrigAddress(String amqOrigAddress) {
        this.amqOrigAddress = amqOrigAddress;
    }

    @JsonProperty("headerCommPattern")
    public String getHeaderCommPattern() {
        return headerCommPattern;
    }

    @JsonProperty("Header_commPattern")
    public void setHeaderCommPattern(String headerCommPattern) {
        this.headerCommPattern = headerCommPattern;
    }

    @JsonProperty("headerMandantId")
    public String getHeaderMandantId() {
        return headerMandantId;
    }

    @JsonProperty("Header_mandantId")
    public void setHeaderMandantId(String headerMandantId) {
        this.headerMandantId = headerMandantId;
    }

    @JsonProperty("headerMessageId")
    public String getHeaderMessageId() {
        return headerMessageId;
    }

    @JsonProperty("Header_messageId")
    public void setHeaderMessageId(String headerMessageId) {
        this.headerMessageId = headerMessageId;
    }

    @JsonProperty("headerMessageType")
    public String getHeaderMessageType() {
        return headerMessageType;
    }

    @JsonProperty("Header_messageType")
    public void setHeaderMessageType(String headerMessageType) {
        this.headerMessageType = headerMessageType;
    }

    @JsonProperty("headerPriority")
    public String getHeaderPriority() {
        return headerPriority;
    }

    @JsonProperty("Header_priority")
    public void setHeaderPriority(String headerPriority) {
        this.headerPriority = headerPriority;
    }

    @JsonProperty("headerReceiver")
    public String getHeaderReceiver() {
        return headerReceiver;
    }

    @JsonProperty("Header_receiver")
    public void setHeaderReceiver(String headerReceiver) {
        this.headerReceiver = headerReceiver;
    }

    @JsonProperty("headerSender")
    public String getHeaderSender() {
        return headerSender;
    }

    @JsonProperty("Header_sender")
    public void setHeaderSender(String headerSender) {
        this.headerSender = headerSender;
    }

    @JsonProperty("headerSenderPlattform")
    public String getHeaderSenderPlattform() {
        return headerSenderPlattform;
    }

    @JsonProperty("Header_senderPlattform")
    public void setHeaderSenderPlattform(String headerSenderPlattform) {
        this.headerSenderPlattform = headerSenderPlattform;
    }

}
