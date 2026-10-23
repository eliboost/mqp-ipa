package ch.postfinance.activemq.monitor;

import org.springframework.context.annotation.Bean;
import org.springframework.jms.support.converter.JacksonJsonMessageConverter;
import org.springframework.jms.support.converter.MessageConverter;
import org.springframework.jms.support.converter.MessageType;

public class Converter {

    @Bean // Serialize message content to json using TextMessage
    public static MessageConverter jacksonJmsMessageConverter() {
      JacksonJsonMessageConverter converter = new JacksonJsonMessageConverter();
      converter.setTargetType(MessageType.TEXT);
      converter.setTypeIdPropertyName("_type");

      return converter;
  }

}
