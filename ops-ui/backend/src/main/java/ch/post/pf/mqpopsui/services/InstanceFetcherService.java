package ch.post.pf.mqpopsui.services;

import ch.post.pf.mqpopsui.config.InstanceFetcherConfig;
import ch.post.pf.mqpopsui.dto.AMQContext;
import ch.post.pf.mqpopsui.dto.AMQInstance;
import ch.post.pf.mqpopsui.security.RolePatternService;
import ch.post.pf.mqpopsui.util.HttpRetry;
import ch.post.pf.mqpopsui.util.InstancesListProperties;
import ch.postfinance.aplat.security.providers.common.cookie.AplatCookieManager;
import com.fasterxml.jackson.dataformat.yaml.YAMLMapper;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.stereotype.Service;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URI;
import java.net.URL;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

@Service
@Slf4j
public class InstanceFetcherService {

    private final InstanceFetcherConfig instanceFetcherConfig;
    private final RolePatternService rolePatternService;
    private final InstancesListProperties instancesListProperties;
    private final YAMLMapper yamlMapper;

    public InstanceFetcherService(
            InstanceFetcherConfig instanceFetcherConfig,
            RolePatternService rolePatternService,
            InstancesListProperties instancesListProperties,
            @Qualifier("yamlMapper") YAMLMapper yamlMapper) {
        this.instanceFetcherConfig = instanceFetcherConfig;
        this.rolePatternService = rolePatternService;
        this.instancesListProperties = instancesListProperties;
        this.yamlMapper = yamlMapper;
    }


    public void fetchInstances() throws Exception {
        String yamlResponse = executeFetchRequest();

        try {
            Map<String, Object> yamlData = yamlMapper.readValue(yamlResponse, Map.class);
            List<AMQContext> contexts = extractContextsFromYaml(yamlData);
            instancesListProperties.setContexts(contexts);
            log.info("Successfully loaded {} contexts", contexts.size());
        } catch (Exception e) {
            log.error("Failed to parse YAML response", e);
            throw new RuntimeException("Failed to parse instance configuration", e);
        }
        AplatCookieManager.getInstance().setRolesPattern(rolePatternService.getRegexPattern());

    }

    private String executeFetchRequest() throws Exception {
        return HttpRetry.execute(instanceFetcherConfig.getMaxRetries(), instanceFetcherConfig.getRetryBackoffMs(), "",
                HttpRetry::isTransient, this::fetchInstanceConfig);
    }

    private HttpURLConnection prepareConnection() throws IOException {
        URL url = URI.create(instanceFetcherConfig.getConfigUrl()).toURL();
        HttpURLConnection connection = (HttpURLConnection) url.openConnection();

        connection.setRequestMethod("GET");

        // Bound the call so a slow/unresponsive GitLab can't hang the request indefinitely.
        connection.setConnectTimeout(instanceFetcherConfig.getConnectTimeoutMs());
        connection.setReadTimeout(instanceFetcherConfig.getReadTimeoutMs());

        return connection;
    }

    private String fetchInstanceConfig() throws IOException {
        HttpURLConnection connection = prepareConnection();
        int responseCode = connection.getResponseCode();
        if (responseCode != 200) {
            throw new RuntimeException("Failed to fetch instances: " + connection.getResponseMessage());
        }

        StringBuilder response = new StringBuilder();
        try (BufferedReader in = new BufferedReader(new InputStreamReader(connection.getInputStream()))) {
            String inputLine;
            while ((inputLine = in.readLine()) != null) {
                response.append(inputLine).append("\n");
            }
        }

        return response.toString();
    }

    private List<AMQContext> extractContextsFromYaml(Map<String, Object> yamlData) {
        List<AMQContext> contexts = new ArrayList<>();

        if (yamlData.containsKey("instances-list")) {
            Map<String, Object> instancesList = (Map<String, Object>) yamlData.get("instances-list");
            List<Map<String, Object>> contextsList = (List<Map<String, Object>>) instancesList.get("contexts");

            for (Map<String, Object> contextData : contextsList) {
                AMQContext context = new AMQContext();
                context.setName((String) contextData.get("name"));

                List<Map<String, Object>> instancesData = (List<Map<String, Object>>) contextData.get("instances");
                List<AMQInstance> instances = instancesData.stream()
                        .map(this::mapToAMQInstance)
                        .toList();
                context.setInstances(instances);

                contexts.add(context);
            }
        }

        return contexts;
    }

    private AMQInstance mapToAMQInstance(Map<String, Object> data) {
        AMQInstance instance = new AMQInstance();
        instance.setName((String) data.get("name"));
        instance.setNamespace((String) data.get("namespace"));
        instance.setReplicas((Integer) data.getOrDefault("replicas", 0));
        instance.setContext((String) data.get("context"));
        instance.setBrokerName((String) data.get("brokerName"));
        instance.setOnline((Boolean) data.getOrDefault("online", false));
        instance.setOperatorRoles((List<String>) data.get("operatorRoles"));
        instance.setQueues((List<String>) data.get("queues"));
        return instance;
    }

}
