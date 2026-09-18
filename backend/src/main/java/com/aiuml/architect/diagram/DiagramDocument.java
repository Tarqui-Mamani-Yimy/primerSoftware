package com.aiuml.architect.diagram;

import com.fasterxml.jackson.annotation.JsonProperty;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import java.util.List;
import java.util.UUID;

/** Renderer-independent, versioned contract. JSON property names intentionally match the frontend. */
public record DiagramDocument(
    @NotNull Integer schemaVersion, UUID id, @NotBlank String name,
    @NotNull List<@Valid UmlClass> classes, @NotNull List<@Valid Relationship> relationships) {
  public record UmlClass(@NotBlank String id, @NotBlank String name, String stereotype, @JsonProperty("package") String packageName,
                         String tableBinding, int x, int y, Integer width, List<Attribute> attributes, List<Method> methods) {}
  public record Attribute(@NotBlank String id, @NotBlank String name, @NotBlank String type, String visibility) {}
  public record Method(@NotBlank String id, @NotBlank String name, @NotBlank String returnType, String visibility) {}
  public record Relationship(@NotBlank String id, @NotBlank String sourceId, @NotBlank String targetId,
                             @NotBlank String type, String sourceMultiplicity, String targetMultiplicity, String label) {}
}
