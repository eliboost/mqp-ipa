package ch.post.pf.mqpopsui.security;

import ch.post.pf.mqpopsui.dto.AMQInstance;
import ch.post.pf.mqpopsui.util.InstancesListProperties;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class RolePatternServiceTest {

    @Mock
    private InstancesListProperties instancesListProperties;

    @InjectMocks
    private RolePatternService rolePatternService;

    @Test
    void shouldBuildRegexFromPrefixes() {
        AMQInstance instance1 = new AMQInstance();
        instance1.setOperatorRoles(List.of("aqs.it_administrator", "asm.mw_operator"));

        AMQInstance instance2 = new AMQInstance();
        instance2.setOperatorRoles(List.of("esbpf.operator"));

        when(instancesListProperties.getAllInstances())
                .thenReturn(List.of(instance1, instance2));

        String pattern = rolePatternService.getRegexPattern();

        assertEquals("^(aqs|asm|esbpf)..*$", pattern);
    }

    @Test
    void shouldHandleDuplicatePrefixes() {
        AMQInstance instance1 = new AMQInstance();
        instance1.setOperatorRoles(List.of("aqs.admin"));

        AMQInstance instance2 = new AMQInstance();
        instance2.setOperatorRoles(List.of("aqs.viewer"));

        when(instancesListProperties.getAllInstances())
                .thenReturn(List.of(instance1, instance2));

        String pattern = rolePatternService.getRegexPattern();

        assertEquals("^(aqs)..*$", pattern);
    }

    @Test
    void shouldIgnoreRolesWithoutDot() {
        AMQInstance instance = new AMQInstance();
        instance.setOperatorRoles(List.of("ADMIN", "aqs.user"));

        when(instancesListProperties.getAllInstances())
                .thenReturn(List.of(instance));

        String pattern = rolePatternService.getRegexPattern();

        assertEquals("^(aqs)..*$", pattern);
    }

    @Test
    void shouldHandleNullRolesList() {
        AMQInstance instance = new AMQInstance();
        instance.setOperatorRoles(null);

        when(instancesListProperties.getAllInstances())
                .thenReturn(List.of(instance));

        String pattern = rolePatternService.getRegexPattern();

        assertEquals("^$", pattern);
    }

    @Test
    void shouldHandleEmptyInstances() {
        when(instancesListProperties.getAllInstances())
                .thenReturn(List.of());

        String pattern = rolePatternService.getRegexPattern();

        assertEquals("^$", pattern);
    }
}