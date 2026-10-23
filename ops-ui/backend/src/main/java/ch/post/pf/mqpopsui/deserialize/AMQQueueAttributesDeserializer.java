package ch.post.pf.mqpopsui.deserialize;

import ch.post.pf.mqpopsui.dto.AMQQueue;
import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.DeserializationContext;
import com.fasterxml.jackson.databind.JsonDeserializer;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.json.JsonMapper;

import java.io.IOException;

public class AMQQueueAttributesDeserializer extends JsonDeserializer<AMQQueue> {
    @Override
    public AMQQueue deserialize(JsonParser p, DeserializationContext ctxt) throws IOException {
        JsonNode node = p.getCodec().readTree(p);
        TypeReference<AMQQueue> typeReferenceList = new TypeReference<>() {
        };
        return new JsonMapper().readValue(node.traverse(), typeReferenceList);

    }
}