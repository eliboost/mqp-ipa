package ch.post.pf.mqpopsui.deserialize;

import ch.post.pf.mqpopsui.dto.jolokia.JolokiaAMQMessage;
import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.DeserializationContext;
import com.fasterxml.jackson.databind.JsonDeserializer;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.json.JsonMapper;

import java.io.IOException;
import java.util.List;

public class AMQMessageDeserializer extends JsonDeserializer<List<JolokiaAMQMessage>> {
    @Override
    public List<JolokiaAMQMessage> deserialize(JsonParser p, DeserializationContext ctxt) throws IOException {
        JsonNode node = p.getCodec().readTree(p);

        if (node.isArray()) {
            TypeReference<List<JolokiaAMQMessage>> typeReferenceList = new TypeReference<>() {
            };
            return new JsonMapper().readValue(node.traverse(), typeReferenceList);
        }
        return null;
    }
}