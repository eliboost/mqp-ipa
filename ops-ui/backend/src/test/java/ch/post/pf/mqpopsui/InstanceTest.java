package ch.post.pf.mqpopsui;

import ch.post.pf.mqpopsui.controller.InstanceController;
import ch.post.pf.mqpopsui.dto.AMQContext;
import ch.post.pf.mqpopsui.dto.AMQInstance;
import ch.post.pf.mqpopsui.security.UserService;
import ch.post.pf.mqpopsui.util.InstancesListProperties;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mockito;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.core.context.SecurityContext;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.test.context.ContextConfiguration;
import org.springframework.test.context.junit.jupiter.SpringExtension;

import java.util.Arrays;
import java.util.Collections;
import java.util.List;

import static org.junit.jupiter.api.Assertions.assertFalse;

@SpringBootTest
@ExtendWith(SpringExtension.class)
@ContextConfiguration
class InstanceTest {

    @Autowired
    private InstanceController instanceController;

    @Autowired
    private InstancesListProperties instancesListProperties;

    @Autowired
    private UserService userService;

    @BeforeEach
    void setUp() {
        // Create a mock Authentication object
        Authentication authentication = Mockito.mock(Authentication.class);
        Mockito.when(authentication.getName()).thenReturn("testUser");

        // Set the authentication in the SecurityContext
        SecurityContext securityContext = SecurityContextHolder.createEmptyContext();
        securityContext.setAuthentication(authentication);
        SecurityContextHolder.setContext(securityContext);

        AMQInstance testInstance = new AMQInstance();
        testInstance.setName("mqp-test");
        testInstance.setOperatorRoles(List.of("ROLE_esbpf.deployer"));
        testInstance.setReplicas(1);
        testInstance.setNamespace("mqp-test-ns");
        testInstance.setContext("e1-pfnet-lab-d");

        AMQContext testContext = new AMQContext();
        testContext.setName("e1-pfnet-lab-d");
        testContext.setInstances(List.of(testInstance));
        instancesListProperties.setContexts(List.of(testContext));
    }

    @Test
    void testLoadingInstancesWorks() {
        // Create a real Authentication object
        Authentication authentication = new UsernamePasswordAuthenticationToken(
                "testUser", // principal
                null, // credentials
                Arrays.asList(
                        new SimpleGrantedAuthority("ROLE_esbpf.deployer"),
                        new SimpleGrantedAuthority("ROLE_middleware.mqp_owner1"),
                        new SimpleGrantedAuthority("ROLE_some_other")
                ) // authorities
        );

        // Set the authentication in the SecurityContext
        SecurityContext securityContext = SecurityContextHolder.createEmptyContext();
        securityContext.setAuthentication(authentication);
        SecurityContextHolder.setContext(securityContext);

        // System.out.println(userService.getUserInfo());

        List<AMQInstance> instances = instanceController.getAllInstances("e1-pfnet-lab-d");
        assertFalse(instances.isEmpty());
    }
}