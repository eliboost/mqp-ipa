package ch.post.pf.mqpopsui.security;


import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.Cookie;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.web.filter.OncePerRequestFilter;

import java.io.IOException;
import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.UUID;

/**
 * Always authenticate the developer-user for every request. Only to be used in a development environment.
 *
 * @author luppim (inspired by hugj and AplatAuthenticationFilter)
 */
@Slf4j
public class DevAuthenticationFilter extends OncePerRequestFilter {

    private static final String SIMULATED_USERNAME_COOKIE_NAME = "dev-simulated-username";
    private static final String SIMULATED_ROLES_COOKIE_NAME = "dev-simulated-roles";

    private static final String DEFAULT_USERNAME = "simulated_user";
    private static final List<String> DEFAULT_ROLES = List.of("ROLE_standard_mqp_ops_ui_role");

    @Override
    protected void doFilterInternal(final HttpServletRequest request, final HttpServletResponse response, final FilterChain filterChain)
            throws ServletException, IOException {
        final String username;
        final List<String> roles = new ArrayList<>();
        final Cookie[] cookies = request.getCookies();
        if (cookies != null) {
            username = Arrays.stream(cookies)
                    .filter(c -> c.getName().equals(SIMULATED_USERNAME_COOKIE_NAME))
                    .map(Cookie::getValue)
                    .findFirst()
                    .orElse(DEFAULT_USERNAME);

            final String rolesAsString = Arrays.stream(cookies)
                    .filter(c -> c.getName().equals(SIMULATED_ROLES_COOKIE_NAME))
                    .map(Cookie::getValue)
                    .map(v -> URLDecoder.decode(v, StandardCharsets.UTF_8))
                    .findFirst()
                    .orElse(StringUtils.join(DEFAULT_ROLES, ","));
            roles.addAll(Arrays.asList(StringUtils.split(rolesAsString, ",")));
        } else {
            username = DEFAULT_USERNAME;
            roles.addAll(DEFAULT_ROLES);
        }

        final Authentication authResult = new UsernamePasswordAuthenticationToken(username, UUID.randomUUID(),
                roles.stream().map(SimpleGrantedAuthority::new).toList());
        SecurityContextHolder.getContext().setAuthentication(authResult);

        log.debug("Authentication success: {}", authResult);

        filterChain.doFilter(request, response);
    }

}
