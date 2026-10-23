package ch.post.pf.mqpopsui.controller;

import ch.post.pf.mqpopsui.dto.jolokia.JolokiaRequest;
import ch.post.pf.mqpopsui.util.InstancesListProperties;
import com.fasterxml.jackson.databind.ObjectMapper;
import io.netty.handler.logging.LogLevel;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.webclient.autoconfigure.WebClientSsl;
import org.springframework.core.ParameterizedTypeReference;
import org.springframework.http.MediaType;
import org.springframework.http.client.reactive.ReactorClientHttpConnector;
import org.springframework.http.codec.json.Jackson2JsonDecoder;
import org.springframework.web.reactive.function.client.WebClient;
import reactor.core.publisher.Mono;
import reactor.netty.http.client.HttpClient;
import reactor.netty.transport.logging.AdvancedByteBufFormat;

import java.net.URI;
import java.util.List;

@Slf4j
public class BaseController {

    private final Object lock = new Object();
    @Autowired
    InstancesListProperties instancesListProperties;
    @Value("${application.platformHasDefaultRole}")
    boolean platformHasDefaultRole;
    @Autowired
    private WebClient.Builder webClientBuilder;
    @Autowired
    private ObjectMapper objectMapper;
    @Autowired
    private WebClientSsl webClientSsl;

    <T> T makeRequest(JolokiaRequest request, URI amqBrokerUri, ParameterizedTypeReference<T> typeReference) {

        // to log all the requests and bodies, uncomment this code as well as line in WebClientBuilder below
        HttpClient httpClient = HttpClient
                .create()
                .wiretap(this.getClass().getCanonicalName(), LogLevel.INFO, AdvancedByteBufFormat.TEXTUAL);

        synchronized (lock) {
            Mono<T> response = webClientBuilder
                    .clientConnector(new ReactorClientHttpConnector(httpClient))
                    .codecs(configurer -> {
                        // Jolokia API returns JSON with content type text/plain, so we need to register a custom
                        // decoder to deserialize this response via Jackson
                        Jackson2JsonDecoder decoder = new Jackson2JsonDecoder(objectMapper, MediaType.TEXT_PLAIN);
                        configurer.customCodecs().registerWithDefaultConfig(decoder);
                    })
                    .apply(webClientSsl.fromBundle("client-bundle"))
                    .build()
                    .post()
                    .uri(amqBrokerUri)
                    .contentType(MediaType.APPLICATION_JSON)
                    .bodyValue(request)
                    .headers(headers -> {
                        headers.setAccept(List.of(MediaType.APPLICATION_JSON));
                        headers.setContentType(MediaType.APPLICATION_JSON);
                    })
                    .retrieve()
                    .bodyToMono(typeReference)
                    .onErrorMap(Exception.class, ex -> {
                        log.error("Error while fetching data from instance {}: {}", amqBrokerUri, ex.getMessage());
                        return new Exception("Error while fetching data from instance", ex);
                    });

            return response.block();
        }
    }

    public void updateProperties(InstancesListProperties newProperties) {
        this.instancesListProperties = newProperties;
    }
}
