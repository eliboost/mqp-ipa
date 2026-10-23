package ch.post.pf.mqpopsui;

import ch.post.pf.mqpopsui.controller.QueueController;
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

@SpringBootTest
@ExtendWith(SpringExtension.class)
@ContextConfiguration
class QueueTest {

    @Autowired
    private QueueController queueController;

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
    }

    @Test
    void addMessageIntoQueue() throws Exception {
        //Create a real authentication object
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

        // ToDo: not working at the moment
        //assertTrue(queueController.sendSingleMessage("e1-mqp-ops-ui-testing-a", "mqp.esbpf.Queue_1", "1234", "SENDER", "RECEIVER", "HelloWorld2", "1", "1234", "1234", "1", "0", "testbody"));
    }
}