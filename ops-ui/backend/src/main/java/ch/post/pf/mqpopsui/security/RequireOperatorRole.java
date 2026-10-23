package ch.post.pf.mqpopsui.security;

import java.lang.annotation.*;

/**
 * Annotation to check if the user has matching operator roles for an instance.
 * If the user doesn't have the required role, a 403 Forbidden response is returned.
 */
@Target(ElementType.METHOD)
@Retention(RetentionPolicy.RUNTIME)
@Documented
public @interface RequireOperatorRole {
    /**
     * The name of the parameter containing the instance name
     */
    String instanceNameParam() default "instanceName";

    /**
     * The name of the parameter containing the context
     */
    String contextParam() default "context";
}

