package com.aiuml.architect.diagram;

import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.regex.Pattern;
import org.springframework.stereotype.Component;

/** Validates semantic invariants that bean validation cannot express across a UML document. */
@Component
class DiagramDocumentValidator {
  private static final Set<String> RELATIONSHIP_TYPES = Set.of(
      "association", "aggregation", "composition", "generalization", "realization", "dependency");
  private static final Pattern MULTIPLICITY = Pattern.compile("\\s*(\\d+|\\*)\\s*(\\.\\.\\s*(\\d+|\\*))?\\s*");

  List<String> validate(DiagramDocument document) {
    var errors = new ArrayList<String>();
    var classesById = new java.util.HashMap<String, DiagramDocument.UmlClass>();
    for (var umlClass : document.classes()) {
      if (classesById.putIfAbsent(umlClass.id(), umlClass) != null) {
        errors.add("Class id must be unique: " + umlClass.id());
      }
    }

    var relationshipIds = new HashSet<String>();
    var relationshipKeys = new HashSet<String>();
    for (var relationship : document.relationships()) {
      if (!relationshipIds.add(relationship.id())) {
        errors.add("Relationship id must be unique: " + relationship.id());
      }
      var source = classesById.get(relationship.sourceId());
      var target = classesById.get(relationship.targetId());
      if (source == null || target == null) {
        errors.add("Relationship " + relationship.id() + " references a missing class");
      }
      if (!RELATIONSHIP_TYPES.contains(relationship.type())) {
        errors.add("Relationship " + relationship.id() + " has unsupported type: " + relationship.type());
      }
      if (relationship.sourceId().equals(relationship.targetId())) {
        errors.add("Relationship " + relationship.id() + " cannot connect a class to itself");
      }
      var key = relationship.sourceId() + "\u0000" + relationship.targetId() + "\u0000" + relationship.type();
      if (!relationshipKeys.add(key)) {
        errors.add("Relationship " + relationship.id() + " duplicates an existing relationship");
      }
      validateMultiplicity(errors, relationship.id(), "source", relationship.sourceMultiplicity());
      validateMultiplicity(errors, relationship.id(), "target", relationship.targetMultiplicity());
      if ("realization".equals(relationship.type()) && source != null && target != null
          && (isInterfaceOrEnum(source) || !"«Interface»".equals(target.stereotype()))) {
        errors.add("Realization requires a non-interface, non-enum source and an interface target");
      }
    }
    return errors;
  }

  private void validateMultiplicity(List<String> errors, String relationshipId, String endpoint, String value) {
    if (value != null && !value.isBlank() && !MULTIPLICITY.matcher(value).matches()) {
      errors.add("Relationship " + relationshipId + " has invalid " + endpoint + " multiplicity");
    }
  }

  private boolean isInterfaceOrEnum(DiagramDocument.UmlClass umlClass) {
    return "«Interface»".equals(umlClass.stereotype()) || "«Enum»".equals(umlClass.stereotype());
  }
}
