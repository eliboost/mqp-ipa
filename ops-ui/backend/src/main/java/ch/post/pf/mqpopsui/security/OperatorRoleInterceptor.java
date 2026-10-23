package ch.post.pf.mqpopsui.security;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import jakarta.servlet.DispatcherType;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;
import org.springframework.web.method.HandlerMethod;
import org.springframework.web.servlet.HandlerInterceptor;

import java.io.IOException;
import java.nio.charset.StandardCharsets;

/**
 * Interceptor that enforces the @RequireOperatorRole annotation.
 *
 * Works for both @RequestParam (reads from query string) and
 * @RequestBody (reads instanceName/context from the JSON body).
 *
 * Requires ContentCachingFilter to be registered so the request body
 * can be read both here and later by the controller.
 */
@Component
@Slf4j
public class OperatorRoleInterceptor implements HandlerInterceptor {

    @Autowired
    private OperatorRoleChecker operatorRoleChecker;

    @Autowired
    private ObjectMapper objectMapper;

    @Override
    public boolean preHandle(HttpServletRequest request, HttpServletResponse response, Object handler) throws Exception {

        if (request.getDispatcherType() == DispatcherType.ASYNC) {
            return true;
        }

        if (!(handler instanceof HandlerMethod handlerMethod)) {
            return true;
        }

        RequireOperatorRole annotation = handlerMethod.getMethodAnnotation(RequireOperatorRole.class);
        if (annotation == null) {
            return true;
        }

        String instanceNameParam = annotation.instanceNameParam();
        String contextParam = annotation.contextParam();

        // Try @RequestParam first (query string / form params)
        String instanceName = request.getParameter(instanceNameParam);
        String context = request.getParameter(contextParam);

        // Fall back to @RequestBody (parse cached JSON body)
        if (instanceName == null || context == null) {
            String[] fromBody = extractFromBody(request, instanceNameParam, contextParam);
            if (instanceName == null) instanceName = fromBody[0];
            if (context == null) context = fromBody[1];
        }

        // Delegate to the shared role checker (throws ResponseStatusException on failure)
        operatorRoleChecker.checkOperatorRole(instanceName, context);

        return true;
    }

    /**
     * Extract field values from the JSON request body.
     * Uses CachedBodyHttpServletRequest so the body can still be read by the controller afterwards.
     */
    private String[] extractFromBody(HttpServletRequest request, String instanceNameField, String contextField) {
        String[] result = {null, null};
        try {
            byte[] bodyBytes = null;

            if (request instanceof ContentCachingFilter.CachedBodyHttpServletRequest cached) {
                bodyBytes = cached.getCachedBody();
            }

            if (bodyBytes == null || bodyBytes.length == 0) {
                return result;
            }

            String body = new String(bodyBytes, StandardCharsets.UTF_8);
            JsonNode node = objectMapper.readTree(body);
            if (node.has(instanceNameField)) {
                result[0] = node.get(instanceNameField).asText(null);
            }
            if (node.has(contextField)) {
                result[1] = node.get(contextField).asText(null);
            }
        } catch (IOException e) {
            log.warn("Could not parse request body for operator role check: {}", e.getMessage());
        }
        return result;
    }
}
