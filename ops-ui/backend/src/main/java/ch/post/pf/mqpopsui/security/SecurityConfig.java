package ch.post.pf.mqpopsui.security;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.config.annotation.method.configuration.EnableMethodSecurity;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.config.annotation.web.configuration.WebSecurityCustomizer;
import org.springframework.security.web.SecurityFilterChain;

@Configuration
@EnableWebSecurity
@EnableMethodSecurity
public class SecurityConfig {

    @Value("${spring.websecurity.debug:false}")
    private boolean webSecurityDebug;

    @Bean
    public WebSecurityCustomizer webSecurityCustomizer() {
        return web -> web.debug(webSecurityDebug);
    }

    @Autowired
    private SecurityAuthenticationConfig securityAuthenticationConfig;

    @Autowired
    private SecurityAuthorisationConfig securityAuthorisationConfig;

    @Autowired
    private SecurityExploitProtectionConfig securityExploitProtectionConfig;

    @Bean
    public SecurityFilterChain MQPOPSUIFilterChain(final HttpSecurity http) throws Exception {

        // -------------------------------------------------------------
        // Authentication Configuration
        // -------------------------------------------------------------

        securityAuthenticationConfig.configure(http);

        // -------------------------------------------------------------
        // Authorisation Configuration
        // -------------------------------------------------------------

        securityAuthorisationConfig.configure(http);

        // -------------------------------------------------------------
        // Exploit Protection Configuration
        // -------------------------------------------------------------

        securityExploitProtectionConfig.configure(http);

        return http.build();
    }
}
