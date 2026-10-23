package ch.post.pf.mqpopsui.security;

import jakarta.servlet.http.HttpServletRequest;
import org.springframework.http.HttpMethod;
import org.springframework.security.web.servlet.util.matcher.PathPatternRequestMatcher;
import org.springframework.security.web.util.matcher.RequestMatcher;

import java.util.Set;

public final class RequestMatcherHelper {

    public static RequestMatcher pathPatternMatcher(final String pattern) {
        return PathPatternRequestMatcher.withDefaults().matcher(pattern);
    }

    public static RequestMatcher httpMethodRequestMatcher(final Set<HttpMethod> httpMethods) {
        return new HttpMethodRequestMatcher(httpMethods);
    }

    private static final class HttpMethodRequestMatcher implements RequestMatcher {

        private final Set<HttpMethod> httpMethods;

        public <T> HttpMethodRequestMatcher(final Set<HttpMethod> httpMethods) {
            this.httpMethods = httpMethods;
        }

        @Override
        public boolean matches(final HttpServletRequest request) {
            return this.httpMethods.contains(HttpMethod.valueOf(request.getMethod()));
        }

        @Override
        public String toString() {
            return "allowedMethods: " + this.httpMethods;
        }

    }

    RequestMatcherHelper() {
    }

}
