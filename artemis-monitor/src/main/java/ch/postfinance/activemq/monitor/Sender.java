package ch.postfinance.activemq.monitor;

import java.util.UUID;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.jms.core.JmsTemplate;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

@Component
@ConditionalOnProperty(name = "application.sender.enabled", havingValue = "true")
public class Sender {

  private final Logger logger = LoggerFactory.getLogger(Sender.class);

  private final JmsTemplate jmsTemplate;
  private final MessageCache cache;


  @Autowired
  public Sender(JmsTemplate jmsTemplate, MessageCache cache,
      @Value("${application.destination}") String destination) {
    this.jmsTemplate = jmsTemplate;
    this.jmsTemplate.setDefaultDestinationName(destination);
    this.jmsTemplate.setMessageConverter(Converter.jacksonJmsMessageConverter());
    this.cache = cache;
  }

  @Scheduled(fixedDelayString = "${application.sender.delay}")
  public void Send() {
    String uuid = UUID.randomUUID().toString();
    logger.debug("Send message: " + uuid);
    MyMessage message = new MyMessage(uuid);

    cache.AddMessage(message);
    this.jmsTemplate.convertAndSend(message);
  }

}
