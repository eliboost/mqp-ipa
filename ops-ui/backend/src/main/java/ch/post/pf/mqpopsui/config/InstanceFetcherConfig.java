package ch.post.pf.mqpopsui.config;

import lombok.Getter;
import lombok.Setter;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.Configuration;

@Configuration
@ConfigurationProperties("application.instance-fetcher")
@Getter
@Setter
public class InstanceFetcherConfig {

    public String configUrl;
    public Integer connectTimeoutMs;
    public Integer readTimeoutMs;
    public Integer maxRetries;
    public Integer retryBackoffMs;

}
