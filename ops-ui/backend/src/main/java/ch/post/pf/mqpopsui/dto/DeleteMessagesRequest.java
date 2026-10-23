package ch.post.pf.mqpopsui.dto;

import lombok.Getter;
import lombok.Setter;

import java.util.List;

@Getter
@Setter
public class DeleteMessagesRequest {
    private String queueName;
    private Integer replica;
    private List<String> messageIds;
    private Integer blockSize;
}
