package ch.post.pf.mqpopsui.security;

import ch.post.pf.mqpopsui.dto.AMQInstance;
import ch.post.pf.mqpopsui.util.InstancesListProperties;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Component;
import org.springframework.web.server.ResponseStatusException;

import java.util.List;
import java.util.Map;

/**
 * Helper service to check operator roles.
 * Can be used to authorize controller method calls before executing business logic.
 * <p>
 * Supports checking authorization from:
 * - Direct parameters: checkOperatorRole(String instanceName, String context)
 * - Request objects with instanceName and context fields: checkOperatorRoleFromRequest(Object request)
 */
@Component
@Slf4j
public class OperatorRoleChecker {

    @Autowired
    private UserService userService;

    @Autowired
    private InstancesListProperties instancesListProperties;

    @Value("${application.platformHasDefaultRole}")
    private boolean platformHasDefaultRole;

    /**
     * Check if the user has the required operator role for the given instance.
     * Throws ResponseStatusException if authorization fails.
     *
     * @param instanceName the name of the instance
     * @param context      the context name
     * @throws ResponseStatusException with 400 if parameters are missing,
     *                                 404 if instance not found,
     *                                 403 if user lacks required role
     */
    public void checkOperatorRole(String instanceName, String context) {
        if (instanceName == null || context == null) {
            throw new ResponseStatusException(HttpStatus.BAD_REQUEST, "Missing required parameters");
        }

        AMQInstance instance = instancesListProperties.getInstanceByNameAndContext(instanceName, context);

        if (instance == null) {
            throw new ResponseStatusException(HttpStatus.NOT_FOUND, "Instance not found");
        }

        if (!hasMatchingRole(userService.getRolesPFPortalAsList(), instance.getOperatorRoles())) {
            Map<String, Object> userInfo = userService.getUserInfo();
            log.warn("User {} attempted to access instance {} without required operator role",
                    userInfo.get("account"), instanceName);
            throw new ResponseStatusException(HttpStatus.FORBIDDEN, "User does not have permission to perform this action");
        }
    }

    /**
     * Check if user roles match instance operator roles.
     */
    private boolean hasMatchingRole(List<String> userRoles, List<String> operatorRoles) {
        List<String> normalizedOperatorRoles = operatorRoles.stream()
                .map(this::normalizeRole)
                .toList();

        return platformHasDefaultRole || userRoles.stream()
                .map(this::normalizeRole)
                .anyMatch(normalizedOperatorRoles::contains);
    }

    private String normalizeRole(String role) {
        return role.replace('.', '_').toLowerCase();
    }
}
