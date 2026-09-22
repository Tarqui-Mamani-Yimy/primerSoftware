import 'models.dart';

/// Returns a detached copy of [document], including all nested UML values.
UmlDocument cloneUmlDocument(UmlDocument document) =>
    UmlDocument.fromJson(document.toJson());

/// Adds an attribute to exactly one class, or returns null without changing
/// [document] when the target or request is invalid.
UmlDocument? addUmlAttribute(
  UmlDocument document, {
  required String className,
  required String attributeId,
  required String attributeName,
  required String attributeType,
  String visibility = '+',
  bool isPk = false,
}) {
  final target = _findUniqueClass(document.classes, className);
  if (target == null ||
      !_validText(attributeId) ||
      !_validText(attributeName) ||
      !_validText(attributeType) ||
      !_validVisibility(visibility) ||
      target.attributes.any(
        (attribute) =>
            attribute.id == attributeId ||
            _sameName(attribute.name, attributeName),
      ) ||
      document.classes
          .expand((umlClass) => umlClass.attributes)
          .any((attribute) => attribute.id == attributeId)) {
    return null;
  }

  final next = cloneUmlDocument(document);
  final nextTarget = _findUniqueClass(next.classes, className)!;
  nextTarget.attributes.add(
    UmlAttribute(
      id: attributeId.trim(),
      name: attributeName.trim(),
      type: attributeType.trim(),
      visibility: visibility,
      isPk: isPk,
    ),
  );
  return next;
}

/// Adds a method to exactly one class, or returns null without changing
/// [document] when the target or request is invalid.
UmlDocument? addUmlMethod(
  UmlDocument document, {
  required String className,
  required String methodId,
  required String methodName,
  required String returnType,
  String visibility = '+',
  bool isAbstract = false,
}) {
  final target = _findUniqueClass(document.classes, className);
  if (target == null ||
      !_validText(methodId) ||
      !_validText(methodName) ||
      !_validText(returnType) ||
      !_validVisibility(visibility) ||
      target.methods.any(
        (method) => method.id == methodId || _sameName(method.name, methodName),
      ) ||
      document.classes
          .expand((umlClass) => umlClass.methods)
          .any((method) => method.id == methodId)) {
    return null;
  }

  final next = cloneUmlDocument(document);
  final nextTarget = _findUniqueClass(next.classes, className)!;
  nextTarget.methods.add(
    UmlMethod(
      id: methodId.trim(),
      name: methodName.trim(),
      returnType: returnType.trim(),
      visibility: visibility,
      isAbstract: isAbstract,
    ),
  );
  return next;
}

/// Creates a relationship between exactly one source and target class.
///
/// Existing relationships with the same endpoints are rejected. The input is
/// never mutated, including when validation fails.
UmlDocument? createUmlRelationship(
  UmlDocument document, {
  required String relationshipId,
  required String sourceClassName,
  required String targetClassName,
  required String type,
  String? sourceMultiplicity,
  String? targetMultiplicity,
  String? label,
}) {
  final source = _findUniqueClass(document.classes, sourceClassName);
  final target = _findUniqueClass(document.classes, targetClassName);
  if (source == null ||
      target == null ||
      source.id == target.id ||
      !_validText(relationshipId) ||
      !_validText(type) ||
      !_optionalText(sourceMultiplicity) ||
      !_optionalText(targetMultiplicity) ||
      !_optionalText(label) ||
      document.relationships.any(
        (relationship) =>
            relationship.id == relationshipId ||
            (relationship.sourceId == source.id &&
                relationship.targetId == target.id),
      )) {
    return null;
  }

  final next = cloneUmlDocument(document);
  next.relationships.add(
    UmlRelationship(
      id: relationshipId.trim(),
      sourceId: source.id,
      targetId: target.id,
      type: type.trim(),
      sourceMultiplicity: _trimOptional(sourceMultiplicity),
      targetMultiplicity: _trimOptional(targetMultiplicity),
      label: _trimOptional(label),
    ),
  );
  return next;
}

/// Deletes exactly one class and every relationship connected to it.
UmlDocument? deleteUmlClass(UmlDocument document, {required String className}) {
  final target = _findUniqueClass(document.classes, className);
  if (target == null) return null;

  final next = cloneUmlDocument(document);
  next.classes.removeWhere((umlClass) => umlClass.id == target.id);
  next.relationships.removeWhere(
    (relationship) =>
        relationship.sourceId == target.id ||
        relationship.targetId == target.id,
  );
  final remainingRelationshipIds = next.relationships
      .map((relationship) => relationship.id)
      .toSet();
  for (var index = 0; index < next.classes.length; index++) {
    final umlClass = next.classes[index];
    if (umlClass.attachedRelationshipId != null &&
        !remainingRelationshipIds.contains(umlClass.attachedRelationshipId)) {
      next.classes[index] = UmlClass(
        id: umlClass.id,
        name: umlClass.name,
        stereotype: umlClass.stereotype,
        package: umlClass.package,
        tableBinding: umlClass.tableBinding,
        x: umlClass.x,
        y: umlClass.y,
        isAssociationClass: umlClass.isAssociationClass,
        attributes: umlClass.attributes,
        methods: umlClass.methods,
      );
    }
  }
  return next;
}

UmlClass? _findUniqueClass(List<UmlClass> classes, String name) {
  if (!_validText(name)) return null;
  final matches = classes
      .where((umlClass) => _sameName(umlClass.name, name))
      .toList(growable: false);
  return matches.length == 1 ? matches.single : null;
}

bool _sameName(String left, String right) =>
    left.trim().toLowerCase() == right.trim().toLowerCase();

bool _validText(String value) => value.trim().isNotEmpty;

bool _optionalText(String? value) => value == null || value.trim().isNotEmpty;

String? _trimOptional(String? value) => value?.trim();

bool _validVisibility(String value) =>
    value == '+' || value == '-' || value == '#';
