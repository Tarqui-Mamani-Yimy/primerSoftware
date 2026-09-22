import 'models.dart';
import 'uml_mutations.dart';
import 'voice_commands.dart';

class VoiceMutationPreview {
  const VoiceMutationPreview({
    required this.command,
    required this.document,
  });

  final VoiceCommand command;
  final UmlDocument document;
}

bool canConfirmVoicePreview(
  VoiceMutationPreview? preview, {
  required bool hasConflict,
}) =>
    preview != null && !hasConflict;

VoiceMutationPreview? invalidateVoicePreview(VoiceMutationPreview? preview) =>
    null;

VoiceMutationPreview? previewVoiceCommand(
  UmlDocument document,
  VoiceCommand command,
) {
  final candidate = _applyVoiceCommand(document, command);
  return candidate == null
      ? null
      : VoiceMutationPreview(command: command, document: candidate);
}

UmlDocument? _applyVoiceCommand(
  UmlDocument document,
  VoiceCommand command,
) {
  switch (command.kind) {
    case VoiceCommandKind.createClass:
      final name = command.name;
      return name == null
          ? null
          : createUmlClass(
              document,
              classId: _idFor('class', name),
              name: name,
            );
    case VoiceCommandKind.addAttribute:
      final className = command.className;
      final name = command.name;
      final type = command.type;
      return className == null || name == null || type == null
          ? null
          : addUmlAttribute(
              document,
              className: className,
              attributeId: _idFor('attribute', '$className-$name'),
              attributeName: name,
              attributeType: type,
            );
    case VoiceCommandKind.addMethod:
      final className = command.className;
      final name = command.name;
      final returnType = command.returnType;
      return className == null || name == null || returnType == null
          ? null
          : addUmlMethod(
              document,
              className: className,
              methodId: _idFor('method', '$className-$name'),
              methodName: name,
              returnType: returnType,
            );
    case VoiceCommandKind.createRelationship:
      final sourceName = command.sourceName;
      final targetName = command.targetName;
      final relationshipType = command.relationshipType;
      return sourceName == null || targetName == null
          ? null
          : createUmlRelationship(
              document,
              relationshipId: _idFor(
                'relationship',
                '$sourceName-$targetName-${relationshipType?.name ?? 'association'}',
              ),
              sourceClassName: sourceName,
              targetClassName: targetName,
              type: relationshipType?.name ?? 'association',
              sourceMultiplicity: command.sourceMultiplicity,
              targetMultiplicity: command.targetMultiplicity,
              label: command.label,
            );
    case VoiceCommandKind.undoVoiceCommand:
      return null;
  }
}

String _idFor(String prefix, String value) {
  final normalized = value
      .trim()
      .toLowerCase()
      .replaceAll(RegExp(r'[^a-z0-9]+'), '-')
      .replaceAll(RegExp(r'^-+|-+$'), '');
  return '$prefix-${normalized.isEmpty ? 'voice' : normalized}';
}
