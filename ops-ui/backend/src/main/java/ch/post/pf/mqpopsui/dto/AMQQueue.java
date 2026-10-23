package ch.post.pf.mqpopsui.dto;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

@Getter
@Setter
@ToString
@JsonIgnoreProperties(ignoreUnknown = true)
public class AMQQueue {

    private String name;
    private String consumerCount;
    private String messageCount;
    private boolean paused;
    private boolean enabled;
    private Integer replica;

    @JsonProperty("name")
    public String getName() {
        return name;
    }

    @JsonProperty("Name")
    public void setName(String name) {
        this.name = name;
    }

    @JsonProperty("consumerCount")
    public String getConsumerCount() {
        return consumerCount;
    }

    @JsonProperty("ConsumerCount")
    public void setConsumerCount(String consumerCount) {
        this.consumerCount = consumerCount;
    }

    @JsonProperty("messageCount")
    public String getMessageCount() {
        return messageCount;
    }

    @JsonProperty("MessageCount")
    public void setMessageCount(String messageCount) {
        this.messageCount = messageCount;
    }

    @JsonProperty("paused")
    public boolean isPaused() {
        return paused;
    }

    @JsonProperty("Paused")
    public void setPaused(boolean paused) {
        this.paused = paused;
    }

    @JsonProperty("enabled")
    public boolean isEnabled() {
        return enabled;
    }

    @JsonProperty("Enabled")
    public void setEnabled(boolean enabled) {
        this.enabled = enabled;
    }

}
