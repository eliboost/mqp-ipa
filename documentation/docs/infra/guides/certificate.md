# Certificate

You will receive a request to create a certificate.

You need to have the following information:
- Hostnames / namespace
- Environment
- Identity Name

We take the following details for this example:
- Hostname: `k8sns-appl-esbpf-e1.e1-k8s-pfnet-a` (k8s) or `e1-esbmsg-alsu011.pnet.ch` (vm)
- Environment: `e1`
- Role: `a-pf-identity-mqp-esbpf-e1`

## Create identity Store in CES

* Storename: `identity-mqp-<user>-<context>`
* Context:    According to cluster env
* E-Mail:     According to teamMail
* Save password: Yes
* Save unencrypted key: Yes 
* Hosts:      As requested by user
* Blueprint:
```
  - Label: node
  - Issuer Type: POSTFINANCE_INTERN
  - Issuer Context: PROD (for e, t -> TEST and p -> PROD)
  - Certificate Type: SIGNING
  - CN: a-pf-identity-mqp-<user>-<context> (or like in the example above  a-pf-identity-mqp-esbpf-e1)
  - Copy Key: Yes
  - Certificate auto-renewal: Yes
```
The user should be the same as the compauth user if the application has one. This certificate is similar to a compauth 
certificate, but additionally includes the context in the CN. This is useful because with compauth you cannot 
distinguish contexts like u1,t1,t2 because they use the same certificate. Within the nucleus area where u1,t1,t2
systems are in the same network, you are able to connect i.e. from t1 to t2.

