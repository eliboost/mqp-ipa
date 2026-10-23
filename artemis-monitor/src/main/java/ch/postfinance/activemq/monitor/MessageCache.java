package ch.postfinance.activemq.monitor;

import com.google.common.cache.Cache;
import com.google.common.cache.CacheBuilder;
import com.google.common.cache.RemovalListener;
import com.google.common.cache.RemovalNotification;
import java.time.Duration;
import java.time.Instant;
import java.util.concurrent.TimeUnit;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

@Component
public class MessageCache implements RemovalListener<String, MyMessage> {

  private final Logger logger = LoggerFactory.getLogger(MessageCache.class);

  private final Metrics metrics;
  private final Cache<String, MyMessage> cache;

  @Autowired
  public MessageCache(Metrics metrics, @Value("${application.message.sla.delay}") Long messageTTL) {
    this.cache = CacheBuilder.newBuilder()
            .expireAfterAccess(Duration.ofMillis(messageTTL))
            .removalListener(this)
            .build();
    this.metrics = metrics;
  }

  public void AddMessage(MyMessage message) {
    metrics.totalMessagesSentCounter.increment();
    cache.put(message.getMessageID(), message);
  }


  public void ClearMessage(String messageID) {
    MyMessage message = cache.getIfPresent(messageID);

    // old or lost message
    if (message == null) {
      return;
    }

    Duration duration = Duration.between(Instant.ofEpochSecond(0, message.getCreatedUtcNs()),
        Instant.now());

    metrics.histogramDuration.record(duration);
    metrics.totalMessagesConsumedCounter.increment();

    cache.invalidate(messageID);
  }

  @Override
  public void onRemoval(RemovalNotification<String, MyMessage> notification) {
    if (!notification.wasEvicted()) {
      return;
    }

    metrics.totalMessagesDelayedCounter.increment();

    logger.debug("Message delayed: " + notification.getValue().getMessageID());
  }
}
