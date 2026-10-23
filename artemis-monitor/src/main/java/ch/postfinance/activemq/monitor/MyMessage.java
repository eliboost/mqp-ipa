package ch.postfinance.activemq.monitor;

import java.io.Serial;
import java.io.Serializable;
import java.time.Instant;

public class MyMessage implements Serializable {
  @Serial
  private static final long serialVersionUID = -295422703255886286L;

  private Long createdUtcNs;
  private String messageID;

  public MyMessage() {}

  public MyMessage(String messageID) {
    Instant instant = Instant.now();

    // Get the epoch second and nanoseconds
    long epochSecond = instant.getEpochSecond();
    int nano = instant.getNano();
    this.createdUtcNs = epochSecond * 1_000_000_000L + nano;
    this.messageID = messageID;
  }

  public Long getCreatedUtcNs() {
    return createdUtcNs;
  }

  public void setCreatedUtcNs(Long createdUtcNs) {
    this.createdUtcNs = createdUtcNs;
  }

  public String getMessageID() {
    return messageID;
  }

  public void setMessageID(String messageID) {
    this.messageID = messageID;
  }

  @Override
  public String toString() {
    return "Message{" +
        "createdUtcNs='" + createdUtcNs + '\'' +
        ", messageID='" + messageID + '\'' +
        '}';
  }
}
