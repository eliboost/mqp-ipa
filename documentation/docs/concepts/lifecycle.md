# Lifecycle

Information about the lifecycle of the Artemis platform, including how CVEs are handled, patching cadence, and
responsibilities.

## Patching
- The Platform Team performs regular, scheduled platform maintenance including image patching of the base images deployed
  by it. That means that the instances are updated by the Platform Team, and the users of the platform do not need to 
  take care of patching.
- Critical CVEs that fall outside the regular cadence are handled as exceptions and patched as quickly as needed.
- Incidents should be rare due to the regular patch cadence, but the team actively monitors scan results and reacts 
whenever issues occur. What to do in case of a CVE is described in the following sections.

## Incident handling
1. Comment the incidents that these are managed by the Platform Team and leave them open until the patch is rolled out.
2. As soon as the rollout is done, the incidents should be closed with the next scan.
3. If not, these can be closed manually with a comment that the patch is rolled out and the next scan will close the incident.

## Rollout, verification & rollback
- All updates follow defined testing and verification steps. Rollback procedures are in place in case of regressions. 
- To meet the RE cycle, the rollout plans take care of RE phases, means no updates near the RE weekend or other critical 
phases of the RE cycle.
- All updates are rolled out env by env, starting with the lab environment, then E‑Platform, T‑Platform, and finally P‑Platform in
  two-week intervals. This allows for verification and rollback if needed before the next environment is updated.

## Communication
- The Platform Team announces platform updates with CVEs, affected versions, mitigation guidance, and rollout schedules via the official 
  blog.
- Release notes will list CVE fixes and any relevant behavioral or configuration changes.
- Subscribe to the [blog](docs/blog/posts/hello-postfinance.md) or the [Community Teams Channel](https://teams.microsoft.com/l/channel/19%3A60da932be53f4dbb81404cbd2d371d1a%40thread.tacv2/ActiveMQ%20Artemis%20Community?groupId=209f33fd-87d6-435a-b4a8-fe3a6f242094&tenantId=3ae7c479-0cf1-47f4-8f84-929f364eff67)
  to get all the necessary information about these updates.
