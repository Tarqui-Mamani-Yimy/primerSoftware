import 'models.dart';

class ImageImportPreview {
  ImageImportPreview._({
    required this.document,
    required this.diagramId,
    required this.generation,
  });

  final UmlDocument document;
  final String? diagramId;
  final int generation;

  factory ImageImportPreview.create({
    required UmlDocument imported,
    required UmlDocument active,
    required int generation,
  }) {
    final json = imported.toJson()..['id'] = active.id;
    final document = UmlDocument.fromJson(json);
    document.version = active.version;
    document.reviewNumber = active.reviewNumber;
    return ImageImportPreview._(
      document: document,
      diagramId: active.id,
      generation: generation,
    );
  }

  bool isCurrent({
    required UmlDocument active,
    required int generation,
  }) =>
      diagramId == active.id && this.generation == generation;

  UmlDocument documentFor(UmlDocument active) {
    final json = document.toJson()..['id'] = active.id;
    final rebound = UmlDocument.fromJson(json);
    rebound.version = active.version;
    rebound.reviewNumber = active.reviewNumber;
    return rebound;
  }
}

bool canConfirmImageImport(
  ImageImportPreview preview, {
  required UmlDocument active,
  required int generation,
  required bool hasConflict,
}) =>
    !hasConflict && preview.isCurrent(active: active, generation: generation);
