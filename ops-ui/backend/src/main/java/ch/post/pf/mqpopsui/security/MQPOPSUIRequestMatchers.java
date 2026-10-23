package ch.post.pf.mqpopsui.security;

import jakarta.servlet.http.HttpServletRequest;
import org.apache.commons.lang3.StringUtils;
import org.springframework.http.HttpMethod;
import org.springframework.security.web.util.matcher.AnyRequestMatcher;
import org.springframework.security.web.util.matcher.RequestMatcher;

import java.util.Arrays;
import java.util.Objects;
import java.util.Set;
import java.util.stream.Collectors;

import static ch.post.pf.mqpopsui.security.RequestMatcherHelper.httpMethodRequestMatcher;
import static ch.post.pf.mqpopsui.security.RequestMatcherHelper.pathPatternMatcher;
import static org.springframework.security.web.util.matcher.RequestMatchers.allOf;
import static org.springframework.security.web.util.matcher.RequestMatchers.anyOf;

public final class MQPOPSUIRequestMatchers {

    private MQPOPSUIRequestMatchers() {
        throw new IllegalStateException("Utility class");
    }

    public static RequestMatcher anyRequestMatcher() {
        return AnyRequestMatcher.INSTANCE;
    }

    public static RequestMatcher noneRequestMatcher() {
        return NoneRequestMatcher.INSTANCE;
    }

    public static RequestMatcher permitAllRequestsByProfile(final String activeProfiles, final String profile) {
        final boolean profileIsActive = Arrays.asList(activeProfiles.split(",")).stream()
                .map(StringUtils::trimToNull)
                .filter(Objects::nonNull)
                .collect(Collectors.toSet())
                .contains(profile);
        if (profileIsActive) {
            return anyRequestMatcher();
        } else {
            return noneRequestMatcher();
        }
    }

    public static RequestMatcher permitAllRequestMatcher() {
        return anyOf(
                pathPatternMatcher("/"),
                pathPatternMatcher("/mqp-ops-ui/actuator/**"),
                pathPatternMatcher("/mqp-ops-ui-nonprod/actuator/**"),
                pathPatternMatcher("/mqp-ops-ui/api/**"),
                pathPatternMatcher("/mqp-ops-ui-nonprod/api/**")
        );
    }

    public static RequestMatcher interactiveRequestMatcher(final Set<HttpMethod> methods) {
        return allOf(
                interactiveRequestMatcher(),
                httpMethodRequestMatcher(methods)
        );
    }

    public static RequestMatcher interactiveRequestMatcher(final HttpMethod... methods) {
        final Set<HttpMethod> httpMethods = Set.of(methods);
        return interactiveRequestMatcher(httpMethods);
    }

    public static RequestMatcher interactiveRequestMatcher() {
        return anyOf(
                pathPatternMatcher("/mqp-ops-ui"),
                pathPatternMatcher("/mqp-ops-ui-nonprod"),
                pathPatternMatcher("/mqp-ops-ui/"),
                pathPatternMatcher("/mqp-ops-ui-nonprod/"),
                pathPatternMatcher("/mqp-ops-ui/**"),
                pathPatternMatcher("/mqp-ops-ui-nonprod/**")
        );
    }

    private static class NoneRequestMatcher implements RequestMatcher {

        public static final RequestMatcher INSTANCE = new NoneRequestMatcher();

        private NoneRequestMatcher() {
        }

        @Override
        public boolean matches(HttpServletRequest request) {
            return false;
        }

        @Override
        @SuppressWarnings("deprecation")
        public boolean equals(Object obj) {
            return obj instanceof NoneRequestMatcher;
        }

        @Override
        public int hashCode() {
            return 1;
        }

        @Override
        public String toString() {
            return "none request";
        }

    }

}
