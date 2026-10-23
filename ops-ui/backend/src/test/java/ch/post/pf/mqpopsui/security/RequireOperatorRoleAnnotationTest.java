package ch.post.pf.mqpopsui.security;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.context.annotation.ClassPathScanningCandidateComponentProvider;
import org.springframework.core.type.filter.AnnotationTypeFilter;
import org.springframework.web.bind.annotation.*;

import java.lang.reflect.Method;
import java.lang.reflect.Parameter;
import java.util.*;
import java.util.stream.Collectors;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("API Security Annotation Tests")
class RequireOperatorRoleAnnotationTest {

    private static final String CONTROLLER_PACKAGE = "ch.post.pf.mqpopsui.controller";
    private static final List<String> EXEMPT_ENDPOINTS = List.of(
            "getAllContexts",
            "getAllInstances",
            "getInstance"
    );

    @Test
    @DisplayName("All API endpoints should have @RequireOperatorRole annotation")
    void allApiEndpointsShouldHaveRequireOperatorRoleAnnotation() {
        List<Method> endpointsWithoutAnnotation = getAllControllerMethods().stream()
                .filter(m -> !EXEMPT_ENDPOINTS.contains(m.getName()))
                .filter(m -> !m.isAnnotationPresent(RequireOperatorRole.class))
                .collect(Collectors.toList());

        String failureMessage = endpointsWithoutAnnotation.isEmpty()
                ? ""
                : "The following endpoints are missing @RequireOperatorRole annotation:\n" +
                endpointsWithoutAnnotation.stream()
                        .map(m -> String.format("  - %s.%s()", m.getDeclaringClass().getSimpleName(), m.getName()))
                        .collect(Collectors.joining("\n"));

        assertTrue(endpointsWithoutAnnotation.isEmpty(), failureMessage);
    }

    @Test
    @DisplayName("All @RequireOperatorRole annotations should have instanceName and context parameters")
    void allAnnotatedMethodsShouldHaveRequiredParameters() {
        Map<String, String> missingParams = new HashMap<>();

        getAllControllerMethods().stream()
                .filter(m -> m.isAnnotationPresent(RequireOperatorRole.class))
                .forEach(method -> {
                    RequireOperatorRole annotation = method.getAnnotation(RequireOperatorRole.class);
                    String instanceNameParam = annotation.instanceNameParam();
                    String contextParam = annotation.contextParam();
                    String methodName = method.getName();

                    List<String> missing = new ArrayList<>();
                    if (!hasParameter(method, instanceNameParam)) {
                        missing.add(instanceNameParam);
                    }
                    if (!hasParameter(method, contextParam)) {
                        missing.add(contextParam);
                    }

                    if (!missing.isEmpty()) {
                        missingParams.put(methodName, "Missing parameters: " + String.join(", ", missing));
                    }
                });

        assertTrue(
                missingParams.isEmpty(),
                "The following methods don't have the required parameters:\n" +
                        missingParams.entrySet().stream()
                                .map(e -> String.format("  - %s: %s", e.getKey(), e.getValue()))
                                .collect(Collectors.joining("\n"))
        );
    }

    @Test
    @DisplayName("All POST/PUT/DELETE endpoints should be protected with @RequireOperatorRole")
    void allMutatingEndpointsShouldBeProtected() {
        List<Method> unprotectedMutatingEndpoints = getAllControllerMethods().stream()
                .filter(this::isMutatingEndpoint)
                .filter(m -> !m.isAnnotationPresent(RequireOperatorRole.class))
                .collect(Collectors.toList());

        String failureMessage = unprotectedMutatingEndpoints.isEmpty()
                ? ""
                : "The following mutating endpoints are not protected with @RequireOperatorRole:\n" +
                unprotectedMutatingEndpoints.stream()
                        .map(m -> String.format("  - %s.%s()", m.getDeclaringClass().getSimpleName(), m.getName()))
                        .collect(Collectors.joining("\n"));

        assertTrue(unprotectedMutatingEndpoints.isEmpty(), failureMessage);
    }

    // Helper methods

    private List<Method> getAllControllerMethods() {
        List<Method> methods = new ArrayList<>();
        ClassPathScanningCandidateComponentProvider provider =
                new ClassPathScanningCandidateComponentProvider(false);
        provider.addIncludeFilter(new AnnotationTypeFilter(RestController.class));

        provider.findCandidateComponents(CONTROLLER_PACKAGE).forEach(beanDef -> {
            try {
                Class<?> clazz = Class.forName(beanDef.getBeanClassName());
                for (Method method : clazz.getDeclaredMethods()) {
                    if (isApiEndpoint(method)) {
                        methods.add(method);
                    }
                }
            } catch (ClassNotFoundException e) {
                fail("Could not load controller class: " + e.getMessage());
            }
        });

        return methods;
    }

    private boolean isApiEndpoint(Method method) {
        return method.isAnnotationPresent(GetMapping.class) ||
                method.isAnnotationPresent(PostMapping.class) ||
                method.isAnnotationPresent(PutMapping.class) ||
                method.isAnnotationPresent(DeleteMapping.class);
    }

    private boolean isMutatingEndpoint(Method method) {
        return method.isAnnotationPresent(PostMapping.class) ||
                method.isAnnotationPresent(PutMapping.class) ||
                method.isAnnotationPresent(DeleteMapping.class);
    }

    private boolean hasParameter(Method method, String paramName) {
        for (Parameter param : method.getParameters()) {
            RequestParam reqParam = param.getAnnotation(RequestParam.class);
            if (reqParam != null && reqParam.value().equals(paramName)) {
                return true;
            }
            if (param.getName().equals(paramName)) {
                return true;
            }
        }
        return false;
    }
}