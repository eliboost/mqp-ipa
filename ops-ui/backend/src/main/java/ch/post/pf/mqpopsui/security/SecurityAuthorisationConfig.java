package ch.post.pf.mqpopsui.security;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;

import static ch.post.pf.mqpopsui.security.MQPOPSUIRequestMatchers.permitAllRequestMatcher;
import static ch.post.pf.mqpopsui.security.MQPOPSUIRequestMatchers.permitAllRequestsByProfile;


/**
 * https://docs.spring.io/spring-security/reference/servlet/authorization/index.html
 */
@Configuration
public class SecurityAuthorisationConfig {

    @Value("${spring.profiles.active:''}")
    private String activeProfiles;

    public void configure(final HttpSecurity http) throws Exception {
        http
                .authorizeHttpRequests(authorize -> authorize
                        .requestMatchers(permitAllRequestsByProfile(activeProfiles, "nonlocal")).permitAll()
                        .requestMatchers(permitAllRequestMatcher()).permitAll()
                        //.requestMatchers(interactiveRequestMatcher()).hasAnyRole(resolveAllRoleNames())
                        .anyRequest().denyAll()
                );
    }
}
