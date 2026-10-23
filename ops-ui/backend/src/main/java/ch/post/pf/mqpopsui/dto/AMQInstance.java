package ch.post.pf.mqpopsui.dto;

import lombok.Getter;
import lombok.Setter;

import java.util.List;

@Getter
@Setter
public class AMQInstance {

    private String name;
    private String namespace;
    private int replicas;
    private String context;
    private String brokerName;
    private boolean online;

    private List<String> operatorRoles;
    private List<String> queues;
}