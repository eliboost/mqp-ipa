package ch.post.pf.mqpopsui.security;

import org.springframework.security.core.Authentication;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.stereotype.Service;

import java.util.*;
import java.util.stream.Collectors;

@Service
public class UserService {

    public Map<String, Object> getUserInfo() {

        final Map<String, Object> userInfoMap = new HashMap<>();
        final Authentication authentication = SecurityContextHolder.getContext().getAuthentication();
        if (authentication != null) {
            userInfoMap.put("account", authentication.getName());
            final Set<String> authorities = authentication.getAuthorities()
                    .stream()
                    .map(GrantedAuthority::getAuthority)
                    .collect(Collectors.toSet());
            userInfoMap.put("rolesPFPortal", authorities);
        }

        return userInfoMap;
    }

    public List<String> getRolesPFPortalAsList() {
        return Arrays.stream(getUserInfo().get("rolesPFPortal").toString()
                        .replaceAll("[\\[\\]]", "") // Remove brackets [ and ]
                        .split(",")) // Split at comma
                .map(String::trim)  // Trim any extra spaces around the values
                .map(role -> role.replace("ROLE_", ""))
                .toList();
    }


}

