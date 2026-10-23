package ch.postfinance.activemq.monitor;

import jakarta.jms.ConnectionFactory;
import java.util.Random;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.boot.jms.autoconfigure.DefaultJmsListenerContainerFactoryConfigurer;
import org.springframework.context.annotation.Bean;
import org.springframework.jms.annotation.JmsListener;
import org.springframework.jms.config.DefaultJmsListenerContainerFactory;
import org.springframework.stereotype.Component;

@Component
@ConditionalOnProperty(name = "application.consumer.enabled", havingValue = "true")
public class Consumer {

  private final Logger logger = LoggerFactory.getLogger(Consumer.class);

  private final MessageCache messageCache;
  private final Metrics metrics;
  private final int minDelay;
  private final int maxDelay;

  @Autowired
  public Consumer(MessageCache messageCache, Metrics metrics,
      @Value("${application.consumer.delay.min}") int minDelay,
      @Value("${application.consumer.delay.max}") int maxDelay
  ) {
    this.messageCache = messageCache;
    this.metrics = metrics;
    this.minDelay = minDelay;
    this.maxDelay = maxDelay;
  }

  @JmsListener(destination = "${application.destination}")
  public void processMessage(MyMessage message) throws InterruptedException {
    messageCache.ClearMessage(message.getMessageID());
    logger.debug("Received message: " + message.getMessageID());

    // imitate process time
    long delay = getRandomDelay();
    Thread.sleep(delay);
    logger.debug("Processed message: {} ({})", message.getMessageID(), delay);
  }

  private long getRandomDelay() {
    if (maxDelay == 0){
      return 0;
    }

    Random rn = new Random();
    int range = maxDelay - minDelay + 1;

    return rn.nextInt(range) + minDelay;
  }

  @Bean
  public DefaultJmsListenerContainerFactory jmsListenerContainerFactory(
      ConnectionFactory connectionFactory, DefaultJmsListenerContainerFactoryConfigurer configurer) {
    DefaultJmsListenerContainerFactory factory = new DefaultJmsListenerContainerFactory();
    configurer.configure(factory, connectionFactory);
    factory.setConnectionFactory(connectionFactory);
    factory.setMessageConverter(Converter.jacksonJmsMessageConverter());
    factory.setErrorHandler(t -> {
      metrics.totalMessagesConsumedErrorsCounter.increment();
      logger.debug("Failed consume: " + t.getMessage());
    });
    // .. other settings
    return factory;
  }
}
