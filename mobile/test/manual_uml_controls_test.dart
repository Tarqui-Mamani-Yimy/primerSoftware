import 'package:ai_uml_architect_mobile/manual_uml_controls.dart';
import 'package:ai_uml_architect_mobile/models.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('shows manual member and relationship controls', (tester) async {
    final document = UmlDocument(
      id: 'diagram-1',
      name: 'Orders',
      classes: [
        UmlClass(
          id: 'user',
          name: 'User',
          attributes: const [
            UmlAttribute(id: 'email', name: 'email', type: 'String'),
          ],
          methods: const [
            UmlMethod(id: 'save', name: 'save', returnType: 'void'),
          ],
        ),
        UmlClass(id: 'order', name: 'Order'),
      ],
    );
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: ManualUmlControls(
              document: document,
              onAddAttribute: (_) {},
              onEditAttribute: (_, __) {},
              onDeleteAttribute: (_, __) {},
              onAddMethod: (_) {},
              onEditMethod: (_, __) {},
              onDeleteMethod: (_, __) {},
              onAddRelationship: () {},
              onEditRelationship: (_) {},
              onDeleteRelationship: (_) {},
            ),
          ),
        ),
      ),
    );

    expect(find.text('Relaciones'), findsOneWidget);
    expect(find.byKey(const Key('manual.relationship.add')), findsOneWidget);
    await tester.tap(find.byKey(const Key('manual.class.user')));
    await tester.pumpAndSettle();
    expect(find.text('Atributos'), findsOneWidget);
    expect(find.text('Métodos'), findsOneWidget);
    expect(find.textContaining('email: String'), findsOneWidget);
    expect(find.textContaining('save(): void'), findsOneWidget);
  });
}
