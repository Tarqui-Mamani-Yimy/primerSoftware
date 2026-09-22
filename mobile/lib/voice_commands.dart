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
    r'^deshacer(?:\s+(?:comando\s+de\s+voz|[úu]ltimo\s+comando\s+de\s+voz))?$';
const _attributePattern =
    r'^(?:agregar|a[ñn]adir)\s+atributo\s+(.+?)\s+de\s+tipo\s+(.+?)\s+a\s+(.+)$';
const _methodPattern =
    r'^(?:agregar|a[ñn]adir)\s+m[ée]todo\s+(.+?)\s+de\s+retorno\s+(.+?)\s+a\s+(.+)$';
final _namePattern = RegExp(r"^[\p{L}\p{N}][\p{L}\p{N}' -]*$", unicode: true);
final _multiplicityPattern = RegExp(
  r'^(?:\d+|\*)(?:\.\.(?:\d+|\*))?$',
);

// Punctuation Deepgram's smart_format inserts inside a transcript (commas,
// semicolons, colons, Spanish inverted marks, and straight/curly/angle
// quotes) that would otherwise break downstream patterns such as
// _cleanName.
final _innerPunctuationPattern = RegExp(r'[,;:¿¡"“”«»]');

/// Strips smart_format punctuation and collapses whitespace before parsing.
String _normalizeTranscript(String raw) {
  final withoutPunctuation = raw.replaceAll(_innerPunctuationPattern, ' ');
  final collapsed = withoutPunctuation.replaceAll(RegExp(r'\s+'), ' ').trim();
  return collapsed.replaceFirst(RegExp(r'[.!?]+$'), '').trim();
}

/// Maps the small Spanish attribute/return type vocabulary (case/accent
/// insensitive) a voice transcript can produce to the canonical Java type
/// name jdlgen expects. Unknown types (including already-canonical English
/// names such as "String") pass through unchanged.
String _mapSpanishType(String value) {
  switch (value.trim().toLowerCase()) {
    case 'entero':
      return 'Integer';
    case 'texto':
    case 'cadena':
      return 'String';
    case 'decimal':
      return 'BigDecimal';
    case 'fecha':
      return 'LocalDate';
    case 'fecha hora':
    case 'fecha y hora':
      return 'ZonedDateTime';
    case 'booleano':
    case 'lógico':
    case 'logico':
      return 'Boolean';
    case 'largo':
      return 'Long';
    case 'flotante':
      return 'Float';
    case 'doble':
      return 'Double';
    case 'uuid':
      return 'UUID';
    default:
      return value;
  }
}

VoiceCommand? parseVoiceCommand(String transcript) {
  final text = _normalizeTranscript(transcript);
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
      type: _mapSpanishType(type),
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
      returnType: _mapSpanishType(returnType),
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

// _spokenNumbers maps the small Spanish cardinal vocabulary a speech-to-text
// transcript can produce for a UML multiplicity bound to its digit form.
const _spokenNumbers = {
  'cero': '0',
  'uno': '1',
  'dos': '2',
  'tres': '3',
  'cuatro': '4',
  'cinco': '5',
  'seis': '6',
  'siete': '7',
  'ocho': '8',
  'nueve': '9',
  'diez': '10',
};

final _digitsOnly = RegExp(r'^\d+$');

/// Resolves one spoken or literal multiplicity bound token ("uno", "5",
/// "muchos", "*") to its digit/"*" form, or null when unrecognized.
String? _spokenBound(String token) {
  final norm = token.trim().toLowerCase();
  if (norm == '*' ||
      norm == 'muchos' ||
      norm == 'varios' ||
      norm == 'asterisco' ||
      norm == 'estrella' ||
      norm == 'n') {
    return '*';
  }
  if (_digitsOnly.hasMatch(norm)) return norm;
  return _spokenNumbers[norm];
}

String? _checkRange(String value) {
  return _multiplicityPattern.hasMatch(value) ? value : null;
}

/// Parses one multiplicity payload into _multiplicityPattern form. Accepts
/// the literal digit/"*" forms plus the spoken Spanish vocabulary a Deepgram
/// transcript produces: a single value ("uno", "muchos"), "X a Y", "de X a
/// Y", "X punto punto Y", the literal "X..Y", "X o más"/"X o mas" ("X..*"),
/// and the idiom "cero o uno" ("0..1").
String? _cleanMultiplicity(String value) {
  final trimmed = value.trim().replaceAll(RegExp(r'\s*\.\.\s*'), '..');
  if (_multiplicityPattern.hasMatch(trimmed)) return trimmed;

  final lower = trimmed.toLowerCase();

  if (RegExp(r'^cero\s+o\s+uno$').hasMatch(lower)) return '0..1';

  final orMore = RegExp(r'^(.+?)\s+o\s+m[aá]s$').firstMatch(lower);
  if (orMore != null) {
    final lo = _spokenBound(orMore.group(1)!);
    return lo == null ? null : _checkRange('$lo..*');
  }

  final deXaY = RegExp(r'^de\s+(.+?)\s+a\s+(.+)$').firstMatch(lower);
  if (deXaY != null) {
    final lo = _spokenBound(deXaY.group(1)!);
    final hi = _spokenBound(deXaY.group(2)!);
    return lo == null || hi == null ? null : _checkRange('$lo..$hi');
  }

  final puntoPunto =
      RegExp(r'^(.+?)\s+punto\s+punto\s+(.+)$').firstMatch(lower);
  if (puntoPunto != null) {
    final lo = _spokenBound(puntoPunto.group(1)!);
    final hi = _spokenBound(puntoPunto.group(2)!);
    return lo == null || hi == null ? null : _checkRange('$lo..$hi');
  }

  final xaY = RegExp(r'^(.+?)\s+a\s+(.+)$').firstMatch(lower);
  if (xaY != null) {
    final lo = _spokenBound(xaY.group(1)!);
    final hi = _spokenBound(xaY.group(2)!);
    return lo == null || hi == null ? null : _checkRange('$lo..$hi');
  }

  final single = _spokenBound(lower);
  return single == null ? null : _checkRange(single);
}
