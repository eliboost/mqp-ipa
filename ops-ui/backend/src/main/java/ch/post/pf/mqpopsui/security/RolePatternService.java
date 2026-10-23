package ch.post.pf.mqpopsui.security;

import ch.post.pf.mqpopsui.dto.AMQInstance;
import ch.post.pf.mqpopsui.util.InstancesListProperties;
import org.springframework.stereotype.Service;

import java.util.List;
import java.util.Objects;
import java.util.stream.Collectors;

@Service
public class RolePatternService {

    private InstancesListProperties instancesListProperties;

    public RolePatternService(InstancesListProperties instancesListProperties) {
        this.instancesListProperties = instancesListProperties;
    }

    public String getRegexPattern() {
        List<AMQInstance> instances = instancesListProperties.getAllInstances();

        String prefixPattern = instances.stream()
                .map(AMQInstance::getOperatorRoles)
                .filter(Objects::nonNull)
                .flatMap(List::stream)
                .filter(Objects::nonNull)
                .map(this::extractPrefix)
                .filter(Objects::nonNull)
                .distinct()
                .collect(Collectors.joining("|"));

        if (prefixPattern.isEmpty()) {
            return "^$";
        }

        return "^(" + prefixPattern + ")..*$";
    }

    private String extractPrefix(String role) {
        int idx = role.indexOf('.');
        if (idx > 0) {
            return role.substring(0, idx);
        }
        return null;
    }

    public void updateProperties(InstancesListProperties newProperties) {
        this.instancesListProperties = newProperties;
    }
}