/*
 * ----------------------------------------------------------------------------
 * Copyright 2009 - 2023 by PostFinance AG - all rights reserved
 * ----------------------------------------------------------------------------
 */
package ch.post.pf.mqpopsui.config;

public enum Platform {

    LOCAL(PlatformType.E),
    E1(PlatformType.E),
    E1NOAUTH(PlatformType.E),
    T1(PlatformType.T),
    T2(PlatformType.T),
    P1(PlatformType.P);

    private final PlatformType type;

    Platform(final PlatformType type) {
        this.type = type;
    }

    public PlatformType getType() {
        return type;
    }

    public enum PlatformType {
        E,
        T,
        P
    }

}
