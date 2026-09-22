enum VoiceCommandKind {
  createClass,
  createRelationship,
  addAttribute,
  addMethod,
  undoVoiceCommand,
}

enum VoiceRelationshipType {
  association,
  aggregation,
  composition,
  generalization,
  realization,
  dependency,
}

class VoiceCommand {
  const VoiceCommand({
    required this.kind,
    this.name,
    this.type,
    this.className,
    this.returnType,
    this.sourceName,
    this.targetName,
    this.relationshipType,
    this.sourceMultiplicity,
    this.targetMultiplicity,
    this.label,
  });

  final VoiceCommandKind kind;
  final String? name;
  final String? type;
  final String? className;
  final String? returnType;
  final String? sourceName;
  final String? targetName;
  final VoiceRelationshipType? relationshipType;
  final String? sourceMultiplicity;
  final String? targetMultiplicity;
  final String? label;
}

const _classPattern = r'^crear\s+(?:una\s+)?clase\s+(.+)$';
const _relationshipPattern = r'^relacionar\s+(.+?)\s+con\s+(.+)$';
const _undoPattern =
    r'^deshacer(?:\s+(?:comando\s+de\s+voz|último\s+comando\s+de\s+voz))?$';
const _attributePattern =
    r'^(?:agregar|añadir)\s+atributo\s+(.+?)\s+de\s+tipo\s+(.+?)\s+a\s+(.+)$';
const _methodPattern =
    r'^(?:agregar|añadir)\s+método\s+(.+?)\s+de\s+retorno\s+(.+?)\s+a\s+(.+)$';
final _namePattern = RegExp(r"^[\p{L}\p{N}][\p{L}\p{N}' -]*$", unicode: true);
final _multiplicityPattern = RegExp(
  r'^(?:\d+|\*)(?:\.\.(?:\d+|\*))?$',
);

VoiceCommand? parseVoiceCommand(String transcript) {
  final text = transcript.trim().replaceFirst(RegExp(r'[.!?]+$'), '').trim();
  if (text.isEmpty ||
      RegExp(r'\bpar[aá]metros?\b', caseSensitive: false).hasMatch(text)) {
    return null;
  }
  if (RegExp(_undoPattern, caseSensitive: false).hasMatch(text)) {
    return const VoiceCommand(kind: VoiceCommandKind.undoVoiceCommand);
  }

  final attribute = RegExp(
    _attributePattern,
    caseSensitive: false,
  ).firstMatch(text);
  if (attribute != null) {
    final name = _cleanName(attribute.group(1)!);
    final type = _cleanName(attribute.group(2)!);
    final className = _cleanName(attribute.group(3)!);
    if (name == null || type == null || className == null) return null;
    return VoiceCommand(
      kind: VoiceCommandKind.addAttribute,
      name: name,
      type: type,
      className: className,
    );
  }

  final method = RegExp(_methodPattern, caseSensitive: false).firstMatch(text);
  if (method != null) {
    final name = _cleanName(method.group(1)!);
    final returnType = _cleanName(method.group(2)!);
    final className = _cleanName(method.group(3)!);
    if (name == null || returnType == null || className == null) return null;
    return VoiceCommand(
      kind: VoiceCommandKind.addMethod,
      name: name,
      returnType: returnType,
      className: className,
    );
  }

  final classMatch = RegExp(
    _classPattern,
    caseSensitive: false,
  ).firstMatch(text);
  if (classMatch != null) {
    final name = _cleanName(classMatch.group(1)!);
    return name == null
        ? null
        : VoiceCommand(kind: VoiceCommandKind.createClass, name: name);
  }

  return _parseRelationship(text);
}

VoiceCommand? _parseRelationship(String text) {
  final match = RegExp(
    _relationshipPattern,
    caseSensitive: false,
  ).firstMatch(text);
  if (match == null) return null;

  final sourceName = _cleanName(match.group(1)!);
  if (sourceName == null) return null;
  final rest = match.group(2)!.trim();
  final markerPattern = RegExp(
    r'\b(como|con\s+cardinalidad|con\s+verbo)\b',
    caseSensitive: false,
  );
  final markers = markerPattern.allMatches(rest).toList();
  final targetRaw =
      markers.isEmpty ? rest : rest.substring(0, markers.first.start);
  final targetName = _cleanName(targetRaw);
  if (targetName == null) return null;

  VoiceRelationshipType? type;
  String? sourceMultiplicity;
  String? targetMultiplicity;
  String? label;
  var hasType = false;
  var hasLabel = false;

  for (var index = 0; index < markers.length; index++) {
    final marker =
        markers[index].group(1)!.toLowerCase().replaceAll(RegExp(r'\s+'), ' ');
    final start = markers[index].end;
    final end =
        index + 1 < markers.length ? markers[index + 1].start : rest.length;
    final payload = rest.substring(start, end).trim();
    if (marker == 'como') {
      if (hasType) return null;
      hasType = true;
      type = _parseRelationshipType(payload);
      if (type == null) return null;
    } else if (marker == 'con cardinalidad') {
      final cardinalities = _parseCardinalities(payload);
      if (cardinalities == null) return null;
      if (cardinalities.$1 != null && sourceMultiplicity != null ||
          cardinalities.$2 != null && targetMultiplicity != null) {
        return null;
      }
      sourceMultiplicity ??= cardinalities.$1;
      targetMultiplicity ??= cardinalities.$2;
    } else if (marker == 'con verbo') {
      if (hasLabel) return null;
      hasLabel = true;
      label = _cleanName(payload);
      if (label == null) return null;
    } else {
      return null;
    }
  }

  return VoiceCommand(
    kind: VoiceCommandKind.createRelationship,
    sourceName: sourceName,
    targetName: targetName,
    relationshipType: type,
    sourceMultiplicity: sourceMultiplicity,
    targetMultiplicity: targetMultiplicity,
    label: label,
  );
}

VoiceRelationshipType? _parseRelationshipType(String value) {
  switch (value.trim().toLowerCase()) {
    case 'asociación':
    case 'asociacion':
    case 'association':
      return VoiceRelationshipType.association;
    case 'agregación':
    case 'agregacion':
    case 'aggregation':
      return VoiceRelationshipType.aggregation;
    case 'composición':
    case 'composicion':
    case 'composition':
      return VoiceRelationshipType.composition;
    case 'generalización':
    case 'generalizacion':
    case 'generalization':
      return VoiceRelationshipType.generalization;
    case 'realización':
    case 'realizacion':
    case 'realization':
      return VoiceRelationshipType.realization;
    case 'dependencia':
    case 'dependency':
      return VoiceRelationshipType.dependency;
    default:
      return null;
  }
}

(String?, String?)? _parseCardinalities(String payload) {
  final bothSourceTarget = RegExp(
    r'^origen\s+(.+?)\s+y\s+(?:cardinalidad\s+)?destino\s+(.+)$',
    caseSensitive: false,
  ).firstMatch(payload);
  if (bothSourceTarget != null) {
    final source = _cleanMultiplicity(bothSourceTarget.group(1)!);
    final target = _cleanMultiplicity(bothSourceTarget.group(2)!);
    return source == null || target == null ? null : (source, target);
  }
  final bothTargetSource = RegExp(
    r'^destino\s+(.+?)\s+y\s+(?:cardinalidad\s+)?origen\s+(.+)$',
    caseSensitive: false,
  ).firstMatch(payload);
  if (bothTargetSource != null) {
    final target = _cleanMultiplicity(bothTargetSource.group(1)!);
    final source = _cleanMultiplicity(bothTargetSource.group(2)!);
    return source == null || target == null ? null : (source, target);
  }
  final sourceOnly = RegExp(
    r'^origen\s+(.+)$',
    caseSensitive: false,
  ).firstMatch(payload);
  if (sourceOnly != null) {
    final source = _cleanMultiplicity(sourceOnly.group(1)!);
    return source == null ? null : (source, null);
  }
  final targetOnly = RegExp(
    r'^destino\s+(.+)$',
    caseSensitive: false,
  ).firstMatch(payload);
  if (targetOnly != null) {
    final target = _cleanMultiplicity(targetOnly.group(1)!);
    return target == null ? null : (null, target);
  }
  return null;
}

String? _cleanName(String value) {
  final name = value.trim().replaceAll(RegExp(r'\s+'), ' ');
  return name.isNotEmpty && name.length <= 80 && _namePattern.hasMatch(name)
      ? name
      : null;
}

String? _cleanMultiplicity(String value) {
  final normalized = value.trim().replaceAll(RegExp(r'\s*\.\.\s*'), '..');
  return _multiplicityPattern.hasMatch(normalized) ? normalized : null;
}
