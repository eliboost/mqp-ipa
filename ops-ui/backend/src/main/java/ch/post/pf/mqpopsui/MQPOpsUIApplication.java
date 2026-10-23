package ch.post.pf.mqpopsui;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.artemis.autoconfigure.ArtemisAutoConfiguration;
import org.springframework.boot.autoconfigure.SpringBootApplication;

@SpringBootApplication(exclude = ArtemisAutoConfiguration.class)
public class MQPOpsUIApplication {


    public static void main(String[] args) {
        SpringApplication.run(MQPOpsUIApplication.class, args);
    }


}