package ch.post.pf.mqpopsui.util;

import ch.post.pf.mqpopsui.dto.AMQInstance;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import org.springframework.web.util.UriComponentsBuilder;

import java.net.URI;
import java.util.regex.Pattern;

@Component
public class AmqEndpointResolver {

    private static final Pattern LABEL = Pattern.compile("[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?");
    private static final String DOMAIN = "pnet.ch";

    private final int jolokiaApiPort;

    public AmqEndpointResolver(@Value("${application.jolokiaApiPort}") int jolokiaApiPort) {
        this.jolokiaApiPort = jolokiaApiPort;
    }

    public void validate(AMQInstance instance, Integer replica) {
        if (instance == null) {
            throw new IllegalArgumentException("Unknown instance");
        }
        if (replica == null || replica < 0 || replica >= instance.getReplicas()) {
            throw new IllegalArgumentException("Invalid replica");
        }
        requireValidLabel(instance.getName(), "name");
        requireValidLabel(instance.getNamespace(), "namespace");
        requireValidLabel(instance.getContext(), "context");
    }

    /** https://<name>-console-<replica>.<ns>.<ctx>.pnet.ch:<port>/console/jolokia */
    public URI consoleUri(AMQInstance instance, Integer replica) {
        validate(instance, replica);
        return UriComponentsBuilder.newInstance()
                .scheme("https")
                .host(host(instance, "console", replica))
                .port(jolokiaApiPort)
                .path("/console/jolokia")
                .build()
                .toUri();
    }

    /** <name>-ssl-<replica>.<ns>.<ctx>.pnet.ch  */
    public String acceptorHost(AMQInstance instance, Integer replica) {
        validate(instance, replica);
        return host(instance, "ssl", replica);
    }

    private String host(AMQInstance instance, String role, int replica) {
        return "%s-%s-%d.%s.%s.%s".formatted(
                instance.getName(), role, replica,
                instance.getNamespace(), instance.getContext(), DOMAIN);
    }

    private static void requireValidLabel(String value, String field) {
        if (value == null || !LABEL.matcher(value).matches()) {
            throw new IllegalArgumentException("Invalid instance configuration: field=" + field + ", value=" + value);
        }
    }
}