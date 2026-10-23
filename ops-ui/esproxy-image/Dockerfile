FROM middleware-docker-local.repo.pnet.ch/pf/mw/esproxy:3.0.202610220841

ARG ESP_CONFIGFILE

COPY --chown=tomcat:tomcat ${ESP_CONFIGFILE} /appl/config/esproxy-config.xml
RUN chmod -R 644 /appl/config/esproxy-config.xml
COPY --chown=tomcat:tomcat start-esproxy.sh /appl/mqp-ops-ui/
RUN chmod +x /appl/mqp-ops-ui/start-esproxy.sh

COPY --chown=tomcat:tomcat cert/truststore.jks /keybox/node-slot/
COPY --chown=tomcat:tomcat cert/password.raw /keybox/signer/
COPY --chown=tomcat:tomcat cert/esproxysigner_keystore.jks /keybox/signer/

CMD ["/bin/sh", "-c", "/appl/mqp-ops-ui/start-esproxy.sh"]