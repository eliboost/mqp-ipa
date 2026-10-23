package ch.post.pf.mqpopsui.util;

import ch.post.pf.mqpopsui.dto.AMQContext;
import ch.post.pf.mqpopsui.dto.AMQInstance;
import lombok.Getter;
import lombok.Setter;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.PropertySource;
import org.springframework.stereotype.Component;

import java.util.ArrayList;
import java.util.List;
import java.util.Optional;

@Setter
@Getter
@Component
public class InstancesListProperties {
    private List<AMQContext> contexts = new ArrayList<>();

    public AMQInstance getInstanceByNameAndContext(String instanceName, String context) {

        Optional<AMQInstance> instance = contexts
                .stream()
                .flatMap(c -> c.getInstances().stream())
                .filter(i -> instanceName.equals(i.getName()) && context.equals(i.getContext()))
                .findAny();

        return instance.orElse(null);
    }

    public List<AMQInstance> getAllInstances() {
        return contexts
                .stream()
                .flatMap(c -> c.getInstances().stream())
                .toList();
    }
}