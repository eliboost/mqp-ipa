package ch.post.pf.mqpopsui.controller;

import ch.post.pf.mqpopsui.dto.AMQInstance;
import ch.post.pf.mqpopsui.security.UserService;
import ch.post.pf.mqpopsui.util.AmqEndpointResolver;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.test.util.ReflectionTestUtils;

import java.net.URI;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class InstanceControllerTest {

    @Mock
    private UserService userService;

    @Mock
    private AmqEndpointResolver amqEndpointResolver;

    @InjectMocks
    private InstanceController instanceController;

    private AMQInstance instanceFromConfig;

    @BeforeEach
    void setUp() {
        instanceFromConfig = new AMQInstance();
        instanceFromConfig.setName("test-instance");
        instanceFromConfig.setOperatorRoles(List.of("aqs_it_administrator"));
        instanceFromConfig.setQueues(List.of("QUEUE.A", "QUEUE.B"));
        instanceFromConfig.setBrokerName("test-broker");
        instanceFromConfig.setNamespace("test-ns");
        instanceFromConfig.setContext("test-ctx");
    }

    @Test
    void shouldReturnNull_whenUserHasNoMatchingRole() {
        when(userService.getRolesPFPortalAsList()).thenReturn(List.of("some_other_role"));
        ReflectionTestUtils.setField(instanceController, "platformHasDefaultRole", false);

        AMQInstance result = invokeGetRealAMQInstance(instanceFromConfig);

        assertThat(result).isNull();
    }

    @Test
    void shouldNotReturnNull_whenUserHasExactRole() {
        when(userService.getRolesPFPortalAsList()).thenReturn(List.of("aqs_it_administrator"));
        when(amqEndpointResolver.consoleUri(instanceFromConfig, 0))
                .thenReturn(URI.create("http://localhost:8161/console/jolokia"));

        ReflectionTestUtils.setField(instanceController, "platformHasDefaultRole", false);

        // Da checkIfInstanceExistsAndGetQueueNames eine HTTP-Anfrage macht, wird hier null zurückgegeben (kein Mock)
        // aber die Rollenprüfung ist bestanden – Instanz wird zurückgegeben (online=false)
        AMQInstance result = invokeGetRealAMQInstance(instanceFromConfig);

        assertThat(result).isNotNull();
        assertThat(result.getName()).isEqualTo("test-instance");
        assertThat(result.isOnline()).isFalse();
    }

    @Test
    void shouldNotReturnNull_whenUserHasRoleWithDotSeparator() {
        // aqs.it_administrator soll auch matchen
        instanceFromConfig.setOperatorRoles(List.of("aqs.it_administrator"));
        when(userService.getRolesPFPortalAsList()).thenReturn(List.of("aqs_it_administrator"));
        ReflectionTestUtils.setField(instanceController, "platformHasDefaultRole", false);

        AMQInstance result = invokeGetRealAMQInstance(instanceFromConfig);

        assertThat(result).isNotNull();
    }

    @Test
    void shouldNotReturnNull_whenPlatformHasDefaultRole() {
        when(userService.getRolesPFPortalAsList()).thenReturn(List.of("totally_unrelated_role"));
        ReflectionTestUtils.setField(instanceController, "platformHasDefaultRole", true);

        AMQInstance result = invokeGetRealAMQInstance(instanceFromConfig);

        assertThat(result).isNotNull();
    }

    @Test
    void shouldReturnNull_whenUserRolesEmpty() {
        when(userService.getRolesPFPortalAsList()).thenReturn(List.of());
        ReflectionTestUtils.setField(instanceController, "platformHasDefaultRole", false);

        AMQInstance result = invokeGetRealAMQInstance(instanceFromConfig);

        assertThat(result).isNull();
    }

    // Hilfsmethode um private Methode aufzurufen
    private AMQInstance invokeGetRealAMQInstance(AMQInstance instance) {
        return (AMQInstance) ReflectionTestUtils.invokeMethod(
                instanceController, "getRealAMQInstanceByInstanceFromConfig", instance
        );
    }
}