package ch.post.pf.mqpopsui.security;

import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.context.SecurityContext;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.web.filter.OncePerRequestFilter;

import java.io.IOException;

/**
 * Stellt sicher dass der in Spring Boot authentisierte Benutzer in den Access Logs des Embedded Tomcat
 * verwendet werden kann.
 * <p>
 * In Tomcat kann danach auf das Request Attribut zugegriffen werden: %{tomcat_accesslog_user}r
 * <p>
 * Folgende Konfiguration ist in application.yml notwendig:
 *
 * <pre>
 * server:
 *   tomcat:
 *     accesslog:
 *       pattern: "%h %l %{tomcat_accesslog_user}r %t %F \"%r\" %s %b"
 * </pre>
 */
public class EmbeddedTomcatAccessLogFilter extends OncePerRequestFilter {

    private static final String REQUEST_ATTRIBUTE_NAME = "tomcat_accesslog_user";

    // Anonymous user is hard-coded in
    // org.springframework.security.web.authentication.AnonymousAuthenticationFilter.AnonymousAuthenticationFilter(java.lang.String)
    private static final String SPRING_SECURITY_ANONYMOUS_USER = "anonymousUser";
    private static final String TOMCAT_ANONYMOUS_USER = "-";

    @Override
    protected void doFilterInternal(final HttpServletRequest request, final HttpServletResponse response, final FilterChain filterChain) throws ServletException, IOException {
        final SecurityContext context = SecurityContextHolder.getContext();
        request.setAttribute(REQUEST_ATTRIBUTE_NAME, extractUserForAccessLog(context));
        filterChain.doFilter(request, response);
    }

    private String extractUserForAccessLog(final SecurityContext context) {
        if (null != context) {
            final Authentication authentication = context.getAuthentication();
            final String authenticatedUserName = authentication.getName();
            if (authentication.isAuthenticated() && !SPRING_SECURITY_ANONYMOUS_USER.equals(authenticatedUserName)) {
                return authenticatedUserName;
            }
        }
        return TOMCAT_ANONYMOUS_USER;
    }

}
