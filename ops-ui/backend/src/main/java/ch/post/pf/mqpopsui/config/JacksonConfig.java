package ch.post.pf.mqpopsui.config;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.json.JsonMapper;
import com.fasterxml.jackson.dataformat.yaml.YAMLMapper;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Primary;

@Configuration
public class JacksonConfig {

    @Bean
    public YAMLMapper yamlMapper() {
        return new YAMLMapper();
    }

    @Bean
    @Primary
    public ObjectMapper customObjectMapper() {
        return JsonMapper.builder().build();
    }
}
