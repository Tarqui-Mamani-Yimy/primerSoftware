import 'package:ai_uml_architect_mobile/image_import_flow.dart';
import 'package:ai_uml_architect_mobile/models.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('binds imported preview to the active diagram baseline', () {
    final active = UmlDocument(
      id: 'diagram-1',
      version: 4,
      reviewNumber: 9,
      name: 'Current',
    );
    final imported = UmlDocument(name: 'Imported', classes: [
      UmlClass(id: 'user', name: 'User'),
    ]);
    final preview = ImageImportPreview.create(
      imported: imported,
      active: active,
      generation: 3,
    );

    expect(preview.document.id, 'diagram-1');
    expect(preview.document.version, 4);
    expect(preview.document.reviewNumber, 9);
    expect(preview.isCurrent(active: active, generation: 3), isTrue);
    expect(preview.isCurrent(active: active, generation: 4), isFalse);
  });

  test('rejects confirmation after a diagram or conflict changes', () {
    final active = UmlDocument(
      id: 'diagram-1',
      version: 4,
      reviewNumber: 9,
      name: 'Current',
    );
    final preview = ImageImportPreview.create(
      imported: UmlDocument(name: 'Imported'),
      active: active,
      generation: 3,
    );

    expect(
      preview.isCurrent(
        active: UmlDocument(id: 'diagram-2', name: 'Other'),
        generation: 3,
      ),
      isFalse,
    );
    expect(
      canConfirmImageImport(preview,
          active: active, generation: 3, hasConflict: true),
      isFalse,
    );
  });

  test('rebinds confirmation to the latest harmless save baseline', () {
    final preview = ImageImportPreview.create(
      imported: UmlDocument(name: 'Imported'),
      active: UmlDocument(
          id: 'diagram-1', version: 4, reviewNumber: 9, name: 'Current'),
      generation: 3,
    );
    final latest = UmlDocument(
      id: 'diagram-1',
      version: 5,
      reviewNumber: 10,
      name: 'Current',
    );

    final confirmed = preview.documentFor(latest);
    expect(confirmed.id, 'diagram-1');
    expect(confirmed.version, 5);
    expect(confirmed.reviewNumber, 10);
  });
}
