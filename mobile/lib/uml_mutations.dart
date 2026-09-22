import 'models.dart';

/// Returns a detached copy of [document], including all nested UML values.
UmlDocument cloneUmlDocument(UmlDocument document) =>
    UmlDocument.fromJson(document.toJson());

/// Adds a class to a detached document, or returns null for invalid or
/// duplicate requests without changing [document].
UmlDocument? createUmlClass(
  UmlDocument document, {
  required String classId,
  required String name,
}) {
  if (!_validText(classId) ||
      !_validText(name) ||
      document.classes.any(
        (umlClass) => umlClass.id == classId || _sameName(umlClass.name, name),
      )) {
    return null;
  }
  final next = cloneUmlDocument(document);
  next.classes.add(UmlClass(id: classId.trim(), name: name.trim()));
  return next;
}

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

UmlDocument? updateUmlAttribute(
  UmlDocument document, {
  required String className,
  required String attributeId,
  required String name,
  required String type,
  String visibility = '+',
  bool isPk = false,
}) {
  final target = _findUniqueClass(document.classes, className);
  if (target == null ||
      !_validText(name) ||
      !_validText(type) ||
      !_validVisibility(visibility)) {
    return null;
  }
  final attribute = target.attributes.where((item) => item.id == attributeId);
  if (attribute.length != 1 ||
      target.attributes.any(
          (item) => item.id != attributeId && _sameName(item.name, name))) {
    return null;
  }
  final next = cloneUmlDocument(document);
  final nextTarget = _findUniqueClass(next.classes, className)!;
  final index =
      nextTarget.attributes.indexWhere((item) => item.id == attributeId);
  nextTarget.attributes[index] = UmlAttribute(
    id: attributeId,
    name: name.trim(),
    type: type.trim(),
    visibility: visibility,
    isPk: isPk,
  );
  return next;
}

UmlDocument? deleteUmlAttribute(
  UmlDocument document, {
  required String className,
  required String attributeId,
}) {
  final target = _findUniqueClass(document.classes, className);
  if (target == null ||
      target.attributes.where((item) => item.id == attributeId).length != 1) {
    return null;
  }
  final next = cloneUmlDocument(document);
  _findUniqueClass(next.classes, className)!
      .attributes
      .removeWhere((item) => item.id == attributeId);
  return next;
}

UmlDocument? updateUmlMethod(
  UmlDocument document, {
  required String className,
  required String methodId,
  required String name,
  required String returnType,
  String visibility = '+',
  bool isAbstract = false,
}) {
  final target = _findUniqueClass(document.classes, className);
  if (target == null ||
      !_validText(name) ||
      !_validText(returnType) ||
      !_validVisibility(visibility)) {
    return null;
  }
  final method = target.methods.where((item) => item.id == methodId);
  if (method.length != 1 ||
      target.methods
          .any((item) => item.id != methodId && _sameName(item.name, name))) {
    return null;
  }
  final next = cloneUmlDocument(document);
  final nextTarget = _findUniqueClass(next.classes, className)!;
  final index = nextTarget.methods.indexWhere((item) => item.id == methodId);
  nextTarget.methods[index] = UmlMethod(
    id: methodId,
    name: name.trim(),
    returnType: returnType.trim(),
    visibility: visibility,
    isAbstract: isAbstract,
  );
  return next;
}

UmlDocument? deleteUmlMethod(
  UmlDocument document, {
  required String className,
  required String methodId,
}) {
  final target = _findUniqueClass(document.classes, className);
  if (target == null ||
      target.methods.where((item) => item.id == methodId).length != 1) {
    return null;
  }
  final next = cloneUmlDocument(document);
  _findUniqueClass(next.classes, className)!
      .methods
      .removeWhere((item) => item.id == methodId);
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
  final normalizedType = type.trim();
  if (source == null ||
      target == null ||
      !_validText(relationshipId) ||
      !_validRelationshipType(normalizedType) ||
      !_optionalText(sourceMultiplicity) ||
      !_optionalText(targetMultiplicity) ||
      !_optionalText(label) ||
      !_validMultiplicity(sourceMultiplicity) ||
      !_validMultiplicity(targetMultiplicity) ||
      document.relationships.any(
        (relationship) =>
            relationship.id == relationshipId ||
            (relationship.sourceId == source.id &&
                relationship.targetId == target.id &&
                relationship.type == normalizedType),
      )) {
    return null;
  }

  final next = cloneUmlDocument(document);
  next.relationships.add(
    UmlRelationship(
      id: relationshipId.trim(),
      sourceId: source.id,
      targetId: target.id,
      type: normalizedType,
      sourceMultiplicity: _trimOptional(sourceMultiplicity),
      targetMultiplicity: _trimOptional(targetMultiplicity),
      label: _trimOptional(label),
    ),
  );
  return next;
}

UmlDocument? updateUmlRelationship(
  UmlDocument document, {
  required String relationshipId,
  required String type,
  String? sourceClassName,
  String? targetClassName,
  String? sourceMultiplicity,
  String? targetMultiplicity,
  String? label,
}) {
  final index = document.relationships
      .indexWhere((relationship) => relationship.id == relationshipId);
  final normalizedType = type.trim();
  final current = index < 0 ? null : document.relationships[index];
  final sourceId = sourceClassName == null
      ? current?.sourceId
      : _findUniqueClass(document.classes, sourceClassName)?.id;
  final targetId = targetClassName == null
      ? current?.targetId
      : _findUniqueClass(document.classes, targetClassName)?.id;
  if (index < 0 ||
      sourceId == null ||
      targetId == null ||
      !_validRelationshipType(normalizedType) ||
      !_optionalText(sourceMultiplicity) ||
      !_optionalText(targetMultiplicity) ||
      !_optionalText(label) ||
      !_validMultiplicity(sourceMultiplicity) ||
      !_validMultiplicity(targetMultiplicity) ||
      document.relationships.asMap().entries.any((entry) =>
          entry.key != index &&
          entry.value.sourceId == sourceId &&
          entry.value.targetId == targetId &&
          entry.value.type == normalizedType)) {
    return null;
  }
  final next = cloneUmlDocument(document);
  final existing = current!;
  next.relationships[index] = UmlRelationship(
    id: existing.id,
    sourceId: sourceId,
    targetId: targetId,
    type: normalizedType,
    sourceMultiplicity: _trimOptional(sourceMultiplicity),
    targetMultiplicity: _trimOptional(targetMultiplicity),
    label: _trimOptional(label),
  );
  return next;
}

UmlDocument? deleteUmlRelationship(
  UmlDocument document, {
  required String relationshipId,
}) {
  if (document.relationships
          .where((item) => item.id == relationshipId)
          .length !=
      1) {
    return null;
  }
  final next = cloneUmlDocument(document);
  next.relationships.removeWhere((item) => item.id == relationshipId);
  for (var index = 0; index < next.classes.length; index++) {
    final umlClass = next.classes[index];
    if (umlClass.attachedRelationshipId == relationshipId) {
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
  final remainingRelationshipIds =
      next.relationships.map((relationship) => relationship.id).toSet();
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

bool _validRelationshipType(String value) => const {
      'association',
      'aggregation',
      'composition',
      'generalization',
      'realization',
      'dependency',
    }.contains(value);

final _multiplicityPattern = RegExp(r'^\s*(\d+|\*)\s*(\.\.\s*(\d+|\*))?\s*$');

bool _validMultiplicity(String? value) =>
    value == null ||
    value.trim().isEmpty ||
    _multiplicityPattern.hasMatch(value);

bool _optionalText(String? value) => value == null || value.trim().isNotEmpty;

String? _trimOptional(String? value) => value?.trim();

bool _validVisibility(String value) =>
    value == '+' || value == '-' || value == '#';
