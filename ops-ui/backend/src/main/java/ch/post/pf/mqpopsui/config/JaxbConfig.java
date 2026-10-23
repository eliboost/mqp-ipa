package ch.post.pf.mqpopsui.config;

import ch.post.pf.mqpopsui.dto.xml.artemis.XmlMessage;
import ch.post.pf.mqpopsui.dto.xml.weblogic.XmlWLJMSMessage;
import jakarta.xml.bind.JAXBContext;
import jakarta.xml.bind.JAXBException;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class JaxbConfig {

    @Bean
    public JAXBContext weblogicJaxbContext() throws JAXBException {
        return JAXBContext.newInstance(XmlWLJMSMessage.class);
    }

    @Bean
    public JAXBContext journalJaxbContext() throws JAXBException {
        return JAXBContext.newInstance(XmlMessage.class);
    }
}
