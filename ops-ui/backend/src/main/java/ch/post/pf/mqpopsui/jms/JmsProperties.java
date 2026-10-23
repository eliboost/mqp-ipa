package ch.post.pf.mqpopsui.jms;

import lombok.Getter;
import lombok.Setter;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.stereotype.Component;

@Component
@ConfigurationProperties("application.jms")
@Getter
@Setter
public class JmsProperties {
    private String trustStorePath;
    private String keyStorePath;
    private int acceptorPort = 7002;
}
