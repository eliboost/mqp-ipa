package ch.post.pf.mqpopsui.security;

import ch.post.pf.mqpopsui.config.Platform;
import ch.postfinance.aplat.security.providers.spring.saml.AplatSamlAuthenticationFilterConfigurator;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Import;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.web.access.intercept.AuthorizationFilter;
import org.springframework.security.web.authentication.www.BasicAuthenticationFilter;

import java.util.EnumSet;

/**
 * https://docs.spring.io/spring-security/reference/servlet/authentication/index.html
 */
@Configuration
@Import({AplatSamlAuthenticationFilterConfigurator.class})
public class SecurityAuthenticationConfig {

    @Value("${application.platform}")
    private Platform platform;

    @Value("${application.aplat.enabled:false}")
    private boolean aplatEnabled;

    @Autowired
    private AplatSamlAuthenticationFilterConfigurator aplatSamlAuthenticationFilterConfigurator;

    @Autowired
    private RolePatternService rolePatternService;

    public void configure(final HttpSecurity http) throws Exception {

        if (aplatEnabled) {
            aplatSamlAuthenticationFilterConfigurator.setAplatRolesPattern(rolePatternService.getRegexPattern());
            aplatSamlAuthenticationFilterConfigurator.configureWebSecurity(http);
        } else if (EnumSet.of(Platform.LOCAL, Platform.E1NOAUTH).contains(platform)) {
            http.addFilterBefore(new DevAuthenticationFilter(), BasicAuthenticationFilter.class);
        }

        http
                .securityContext(securityContext -> securityContext.requireExplicitSave(true))
                .addFilterBefore(new EmbeddedTomcatAccessLogFilter(), AuthorizationFilter.class);
    }

}
