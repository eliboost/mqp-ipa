package ch.post.pf.mqpopsui.controller;

import ch.post.pf.mqpopsui.dto.AMQContext;
import ch.post.pf.mqpopsui.dto.AMQInstance;
import ch.post.pf.mqpopsui.dto.jolokia.JolokiaRequest;
import ch.post.pf.mqpopsui.dto.jolokia.JolokiaStringResponse;
import ch.post.pf.mqpopsui.security.UserService;
import ch.post.pf.mqpopsui.util.AmqEndpointResolver;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.core.ParameterizedTypeReference;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.net.URI;
import java.util.List;
import java.util.Objects;
import java.util.stream.Collectors;
import java.util.stream.Stream;

@RestController
@RequestMapping({"/mqp-ops-ui/api", "/mqp-ops-ui-nonprod/api"})
@Slf4j
public class InstanceController extends BaseController {

    @Autowired
    private UserService userService;
    @Autowired
    private AmqEndpointResolver amqEndpointResolver;

    @GetMapping("/v1/contexts")
    public List<AMQContext> getAllContexts() {
        return this.instancesListProperties.getContexts();
    }

    @GetMapping("/v1/instances")
    public List<AMQInstance> getAllInstances(@RequestParam() String context) {
        return this.instancesListProperties.getContexts().stream()
                .filter(c -> c.getName().equals(context))
                .flatMap(c -> c.getInstances().stream())
                // Map to the real AMQ instance
                .map(this::getRealAMQInstanceByInstanceFromConfig)
                // Filter out null instances
                .filter(Objects::nonNull)
                .collect(Collectors.toList());
    }

    @GetMapping("/v1/instance")
    public AMQInstance getInstance(@RequestParam() String instanceToFind, @RequestParam() String context) {
        return this.instancesListProperties.getContexts().stream()
                .flatMap(c -> c.getInstances().stream())
                .filter(instanceFromConfig -> context.equals(instanceFromConfig.getContext()))
                // Find the matching instance
                .filter(instanceFromConfig -> instanceToFind.equals(instanceFromConfig.getName()))
                // Map to the real AMQ instance
                .map(this::getRealAMQInstanceByInstanceFromConfig)
                // Filter out null values
                .filter(Objects::nonNull)
                // Return the first matching instance, if present
                .findAny().orElse(null);
    }

    private AMQInstance getRealAMQInstanceByInstanceFromConfig(AMQInstance instanceFromConfig) {
        if (userService.getRolesPFPortalAsList().stream().anyMatch(r ->
                platformHasDefaultRole ||
                        instanceFromConfig.getOperatorRoles().stream().anyMatch(operatorRole ->
                                normalizeRole(r).equals(normalizeRole(operatorRole))
                        )
        )) {
            // Get real data from instance and generate an instance object holding the name and the
            // queues according to the self-service config
            List<String> queueNamesMatchingRealInstanceAndConfig = checkIfInstanceExistsAndGetQueueNames(instanceFromConfig);

            AMQInstance i = new AMQInstance();
            i.setName(instanceFromConfig.getName());
            i.setReplicas(instanceFromConfig.getReplicas());
            i.setContext(instanceFromConfig.getContext());
            i.setOperatorRoles(instanceFromConfig.getOperatorRoles());

            if (queueNamesMatchingRealInstanceAndConfig != null) {
                i.setQueues(queueNamesMatchingRealInstanceAndConfig);
                i.setOnline(true);
            } else {
                i.setOnline(false);
            }
            return i;
        }
        return null;
    }

    private String normalizeRole(String role) {
        return role.replace('.', '_').toLowerCase();
    }

    private List<String> checkIfInstanceExistsAndGetQueueNames(AMQInstance instance) {
        JolokiaRequest request = new JolokiaRequest();
        request.setType("EXEC");
        request.setMbean("org.apache.activemq.artemis:broker=\"" + instance.getBrokerName() + "\"");
        request.setOperation("getQueueNames");
        request.setArguments(new String[]{"ANYCAST"});
        
        URI amqBrokerUri = amqEndpointResolver.consoleUri(instance, 0);
        
        try {

            ParameterizedTypeReference<JolokiaStringResponse> typeReference = new ParameterizedTypeReference<>() {
            };
            JolokiaStringResponse resp = makeRequest(request, amqBrokerUri, typeReference);

            if (resp.getValue() != null) {
                List<String> queuesFromRealInstance = resp.getValue();

                return queuesFromRealInstance.stream()
                        .filter(q -> instance.getQueues().contains(q))
                        .flatMap(q -> {
                            if (queuesFromRealInstance.contains(q + "_DLQ")) {
                                return Stream.of(q, q + "_DLQ");
                            }
                            return Stream.of(q);
                        })
                        .flatMap(q -> {
                            if (queuesFromRealInstance.contains(q + "_EXPIRY")) {
                                return Stream.of(q, q + "_EXPIRY");
                            }
                            return Stream.of(q);
                        })
                        .sorted()
                        .toList();
            }
        } catch (Exception e) {
            log.error("Error with an instance {}.", e.getMessage());
        }

        return null;
    }

}
