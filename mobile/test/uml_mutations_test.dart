import 'package:ai_uml_architect_mobile/models.dart';
import 'package:ai_uml_architect_mobile/uml_mutations.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('creates a class without mutating the input', () {
    final original = _document();
    final updated = createUmlClass(
      original,
      classId: 'invoice',
      name: 'Invoice',
    );

    expect(updated, isNotNull);
    expect(updated!.classes.map((item) => item.name), contains('Invoice'));
    expect(original.classes, hasLength(2));
  });

  test('clone creates a detached document and preserves baselines', () {
    final original = _document();
    final copy = cloneUmlDocument(original);

    expect(copy.toJson(), equals(original.toJson()));
    copy.classes.first.name = 'Changed';
    copy.classes.first.attributes.add(
      const UmlAttribute(id: 'new', name: 'newField', type: 'String'),
    );

    expect(original.classes.first.name, 'User');
    expect(original.classes.first.attributes, hasLength(1));
    expect(copy.version, 7);
    expect(copy.reviewNumber, 11);
  });

  test('adds an attribute without mutating the input', () {
    final original = _document();
    final updated = addUmlAttribute(
      original,
      className: 'User',
      attributeId: 'email',
      attributeName: 'email',
      attributeType: 'String',
    );

    expect(updated, isNotNull);
    expect(
      updated!.classes.first.attributes.map((item) => item.name),
      containsAll(<String>['id', 'email']),
    );
    expect(original.classes.first.attributes, hasLength(1));
  });

  test('adds a method without mutating the input', () {
    final original = _document();
    final updated = addUmlMethod(
      original,
      className: 'User',
      methodId: 'validate',
      methodName: 'validate',
      returnType: 'bool',
    );

    expect(updated, isNotNull);
    expect(updated!.classes.first.methods.single.returnType, 'bool');
    expect(original.classes.first.methods, isEmpty);
  });

  test('edits and deletes an attribute without mutating input', () {
    final original = _document();
    final edited = updateUmlAttribute(
      original,
      className: 'User',
      attributeId: 'id',
      name: 'userId',
      type: 'String',
      visibility: '-',
    );

    expect(edited, isNotNull);
    expect(edited!.classes.first.attributes.single.name, 'userId');
    expect(edited.classes.first.attributes.single.type, 'String');
    expect(original.classes.first.attributes.single.name, 'id');

    final deleted = deleteUmlAttribute(
      edited,
      className: 'User',
      attributeId: 'id',
    );
    expect(deleted, isNotNull);
    expect(deleted!.classes.first.attributes, isEmpty);
    expect(edited.classes.first.attributes, hasLength(1));
  });

  test('edits and deletes a method without mutating input', () {
    final original = _document(
      methods: const [
        UmlMethod(id: 'validate', name: 'validate', returnType: 'bool'),
      ],
    );
    final edited = updateUmlMethod(
      original,
      className: 'User',
      methodId: 'validate',
      name: 'check',
      returnType: 'String',
      visibility: '#',
    );

    expect(edited, isNotNull);
    expect(edited!.classes.first.methods.single.name, 'check');
    expect(original.classes.first.methods.single.name, 'validate');

    final deleted = deleteUmlMethod(
      edited,
      className: 'User',
      methodId: 'validate',
    );
    expect(deleted, isNotNull);
    expect(deleted!.classes.first.methods, isEmpty);
  });

  test('edits and deletes a relationship without mutating input', () {
    final original = _document(
      relationships: const [
        UmlRelationship(
          id: 'user-order',
          sourceId: 'user',
          targetId: 'order',
          type: 'association',
        ),
      ],
    );
    final edited = updateUmlRelationship(
      original,
      relationshipId: 'user-order',
      type: 'dependency',
      sourceMultiplicity: '1',
      targetMultiplicity: '*',
      label: 'uses',
    );

    expect(edited, isNotNull);
    expect(edited!.relationships.single.type, 'dependency');
    expect(original.relationships.single.type, 'association');

    final deleted = deleteUmlRelationship(edited, relationshipId: 'user-order');
    expect(deleted, isNotNull);
    expect(deleted!.relationships, isEmpty);
    expect(edited.relationships, hasLength(1));
  });

  test('creates a validated relationship and preserves baselines', () {
    final original = _document();
    final updated = createUmlRelationship(
      original,
      relationshipId: 'user-order',
      sourceClassName: 'User',
      targetClassName: 'Order',
      type: 'association',
      targetMultiplicity: '1..*',
    );

    expect(updated, isNotNull);
    expect(updated!.relationships.single.sourceId, 'user');
    expect(updated.relationships.single.targetId, 'order');
    expect(updated.version, original.version);
    expect(updated.reviewNumber, original.reviewNumber);
    expect(original.relationships, isEmpty);
  });

  test('rejects an identical relationship without mutating input', () {
    final original = _document(
      relationships: const [
        UmlRelationship(
          id: 'user-order',
          sourceId: 'user',
          targetId: 'order',
          type: 'association',
        ),
      ],
    );
    final before = original.toJson();

    expect(
      createUmlRelationship(
        original,
        relationshipId: 'another-id',
        sourceClassName: 'User',
        targetClassName: 'Order',
        type: 'association',
      ),
      isNull,
    );
    expect(original.toJson(), equals(before));
  });

  test('allows the same endpoints with a different valid type', () {
    final original = _document(
      relationships: const [
        UmlRelationship(
          id: 'user-order',
          sourceId: 'user',
          targetId: 'order',
          type: 'association',
        ),
      ],
    );

    final updated = createUmlRelationship(
      original,
      relationshipId: 'user-order-dependency',
      sourceClassName: 'User',
      targetClassName: 'Order',
      type: 'dependency',
    );

    expect(updated, isNotNull);
    expect(updated!.relationships, hasLength(2));
    expect(updated.relationships.last.type, 'dependency');
    expect(original.relationships, hasLength(1));
  });

  test('deletes a class and all connected relationships', () {
    final original = _document(
      relationships: const [
        UmlRelationship(
          id: 'user-order',
          sourceId: 'user',
          targetId: 'order',
          type: 'association',
        ),
        UmlRelationship(
          id: 'order-user',
          sourceId: 'order',
          targetId: 'user',
          type: 'dependency',
        ),
      ],
    );
    final updated = deleteUmlClass(original, className: 'User');

    expect(updated, isNotNull);
    expect(updated!.classes.map((item) => item.name), ['Order']);
    expect(updated.relationships, isEmpty);
    expect(original.classes, hasLength(2));
    expect(original.relationships, hasLength(2));
  });

  test(
    'rejects unsupported types and invalid endpoint multiplicities unchanged',
    () {
      final original = _document();
      final before = original.toJson();

      expect(
        createUmlRelationship(
          original,
          relationshipId: 'invalid-type',
          sourceClassName: 'User',
          targetClassName: 'Order',
          type: 'realizes',
        ),
        isNull,
      );
      expect(
        createUmlRelationship(
          original,
          relationshipId: 'invalid-source-multiplicity',
          sourceClassName: 'User',
          targetClassName: 'Order',
          type: 'association',
          sourceMultiplicity: 'many',
        ),
        isNull,
      );
      expect(
        createUmlRelationship(
          original,
          relationshipId: 'invalid-target-multiplicity',
          sourceClassName: 'User',
          targetClassName: 'Order',
          type: 'association',
          targetMultiplicity: '1..many',
        ),
        isNull,
      );
      expect(original.toJson(), equals(before));
    },
  );

  test('allows a valid recursive relationship', () {
    final original = _document();
    final updated = createUmlRelationship(
      original,
      relationshipId: 'user-parent',
      sourceClassName: 'User',
      targetClassName: 'User',
      type: 'association',
      sourceMultiplicity: '0..1',
      targetMultiplicity: '*',
    );

    expect(updated, isNotNull);
    expect(updated!.relationships.single.sourceId, 'user');
    expect(updated.relationships.single.targetId, 'user');
    expect(original.relationships, isEmpty);
  });

  test(
    'rejects missing, ambiguous, duplicate, and invalid requests unchanged',
    () {
      final original = _document(
        duplicateUser: true,
        relationships: const [
          UmlRelationship(
            id: 'existing',
            sourceId: 'user',
            targetId: 'order',
            type: 'association',
          ),
        ],
      );
      final before = original.toJson();

      expect(
        addUmlAttribute(
          original,
          className: 'Missing',
          attributeId: 'x',
          attributeName: 'x',
          attributeType: 'String',
        ),
        isNull,
      );
      expect(
        addUmlMethod(
          original,
          className: 'User',
          methodId: 'x',
          methodName: 'x',
          returnType: 'void',
        ),
        isNull,
      );
      expect(
        createUmlRelationship(
          original,
          relationshipId: 'new',
          sourceClassName: 'User',
          targetClassName: 'Order',
          type: 'association',
        ),
        isNull,
      );
      expect(deleteUmlClass(original, className: 'User'), isNull);
      expect(original.toJson(), equals(before));
    },
  );
}

UmlDocument _document({
  List<UmlRelationship> relationships = const [],
  bool duplicateUser = false,
  List<UmlMethod> methods = const [],
}) {
  final users = <UmlClass>[
    UmlClass(
      id: 'user',
      name: 'User',
      attributes: [
        const UmlAttribute(id: 'id', name: 'id', type: 'UUID', isPk: true),
      ],
      methods: methods,
    ),
  ];
  if (duplicateUser) {
    users.add(UmlClass(id: 'user-copy', name: 'User'));
  }
  return UmlDocument(
    id: 'diagram-1',
    version: 7,
    reviewNumber: 11,
    name: 'Orders',
    classes: [
      ...users,
      UmlClass(id: 'order', name: 'Order'),
    ],
    relationships: relationships,
  );
}
