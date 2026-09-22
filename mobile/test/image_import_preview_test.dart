import 'package:ai_uml_architect_mobile/image_import_preview.dart';
import 'package:ai_uml_architect_mobile/models.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter/material.dart';

void main() {
  testWidgets('shows explicit image import confirmation and cancellation',
      (tester) async {
    var confirmed = false;
    var cancelled = false;
    await tester.pumpWidget(MaterialApp(
      home: Scaffold(
        body: ImageImportPreviewPanel(
          document: UmlDocument(
            name: 'Imported',
            classes: [UmlClass(id: 'user', name: 'User')],
          ),
          onConfirm: () => confirmed = true,
          onCancel: () => cancelled = true,
        ),
      ),
    ));

    expect(find.byKey(const Key('image-import.preview')), findsOneWidget);
    await tester.tap(find.byKey(const Key('image-import.confirm')));
    await tester.tap(find.byKey(const Key('image-import.cancel')));
    expect(confirmed, isTrue);
    expect(cancelled, isTrue);
  });
}
