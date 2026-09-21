class UmlAttribute {
  const UmlAttribute({
    required this.id,
    required this.name,
    required this.type,
    this.visibility = '+',
    this.isPk = false,
  });

  final String id;
  final String name;
  final String type;
  final String visibility;
  final bool isPk;

  factory UmlAttribute.fromJson(Map<String, dynamic> json) => UmlAttribute(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        type: json['type'] as String? ?? 'String',
        visibility: json['visibility'] as String? ?? '+',
        isPk: json['isPk'] as bool? ?? false,
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'type': type,
        'visibility': visibility,
        'isPk': isPk,
      };
}

class UmlMethod {
  const UmlMethod({
    required this.id,
    required this.name,
    required this.returnType,
    this.visibility = '+',
    this.isAbstract = false,
  });

  final String id;
  final String name;
  final String returnType;
  final String visibility;
  final bool isAbstract;

  factory UmlMethod.fromJson(Map<String, dynamic> json) => UmlMethod(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        returnType: json['returnType'] as String? ?? 'void',
        visibility: json['visibility'] as String? ?? '+',
        isAbstract: json['isAbstract'] as bool? ?? false,
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'returnType': returnType,
        'visibility': visibility,
        'isAbstract': isAbstract,
      };
}

class UmlClass {
  UmlClass({
    required this.id,
    required this.name,
    this.stereotype = '«Entity»',
    this.package = '',
    this.tableBinding = '',
    this.x = 80,
    this.y = 80,
    this.isAssociationClass = false,
    this.attachedRelationshipId,
    List<UmlAttribute>? attributes,
    List<UmlMethod>? methods,
  })  : attributes = attributes ?? <UmlAttribute>[],
        methods = methods ?? <UmlMethod>[];

  final String id;
  String name;
  final String stereotype;
  final String package;
  final String tableBinding;
  final double x;
  final double y;
  bool isAssociationClass;
  final String? attachedRelationshipId;
  final List<UmlAttribute> attributes;
  final List<UmlMethod> methods;

  factory UmlClass.fromJson(Map<String, dynamic> json) => UmlClass(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? 'Unnamed',
        stereotype: json['stereotype'] as String? ?? '«Entity»',
        package: json['package'] as String? ?? '',
        tableBinding: json['tableBinding'] as String? ?? '',
        x: (json['x'] as num?)?.toDouble() ?? 80,
        y: (json['y'] as num?)?.toDouble() ?? 80,
        isAssociationClass: json['isAssociationClass'] as bool? ?? false,
        attachedRelationshipId: json['attachedRelationshipId'] as String?,
        attributes: ((json['attributes'] as List?) ?? const [])
            .map((item) => UmlAttribute.fromJson(item as Map<String, dynamic>))
            .toList(),
        methods: ((json['methods'] as List?) ?? const [])
            .map((item) => UmlMethod.fromJson(item as Map<String, dynamic>))
            .toList(),
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'stereotype': stereotype,
        'package': package,
        'tableBinding': tableBinding,
        'x': x,
        'y': y,
        'isAssociationClass': isAssociationClass,
        if (attachedRelationshipId != null) 'attachedRelationshipId': attachedRelationshipId,
        'attributes': attributes.map((attribute) => attribute.toJson()).toList(),
        'methods': methods.map((method) => method.toJson()).toList(),
      };
}

class UmlRelationship {
  const UmlRelationship({
    required this.id,
    required this.sourceId,
    required this.targetId,
    required this.type,
    this.sourceMultiplicity,
    this.targetMultiplicity,
    this.label,
  });

  final String id;
  final String sourceId;
  final String targetId;
  final String type;
  final String? sourceMultiplicity;
  final String? targetMultiplicity;
  final String? label;

  factory UmlRelationship.fromJson(Map<String, dynamic> json) => UmlRelationship(
        id: json['id'] as String? ?? '',
        sourceId: json['sourceId'] as String? ?? '',
        targetId: json['targetId'] as String? ?? '',
        type: json['type'] as String? ?? 'association',
        sourceMultiplicity: json['sourceMultiplicity'] as String?,
        targetMultiplicity: json['targetMultiplicity'] as String?,
        label: json['label'] as String?,
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        'sourceId': sourceId,
        'targetId': targetId,
        'type': type,
        if (sourceMultiplicity != null) 'sourceMultiplicity': sourceMultiplicity,
        if (targetMultiplicity != null) 'targetMultiplicity': targetMultiplicity,
        if (label != null) 'label': label,
      };
}

class UmlDocument {
  UmlDocument({
    this.schemaVersion = 1,
    this.id,
    this.version = 0,
    this.reviewNumber = 0,
    required this.name,
    List<UmlClass>? classes,
    List<UmlRelationship>? relationships,
  })  : classes = classes ?? <UmlClass>[],
        relationships = relationships ?? <UmlRelationship>[];

  final int schemaVersion;
  final String? id;
  int version;
  int reviewNumber;
  String name;
  final List<UmlClass> classes;
  final List<UmlRelationship> relationships;

  factory UmlDocument.fromJson(Map<String, dynamic> json) => UmlDocument(
        schemaVersion: json['schemaVersion'] as int? ?? 1,
        id: json['id'] as String?,
        version: (json['version'] as num?)?.toInt() ?? 0,
        reviewNumber: (json['reviewNumber'] as num?)?.toInt() ?? 0,
        name: json['name'] as String? ?? 'Untitled diagram',
        classes: ((json['classes'] as List?) ?? const [])
            .map((item) => UmlClass.fromJson(item as Map<String, dynamic>))
            .toList(),
        relationships: ((json['relationships'] as List?) ?? const [])
            .map((item) => UmlRelationship.fromJson(item as Map<String, dynamic>))
            .toList(),
      );

  Map<String, dynamic> toJson() => {
        'schemaVersion': schemaVersion,
        if (id != null) 'id': id,
        'version': version,
        'reviewNumber': reviewNumber,
        'name': name,
        'classes': classes.map((item) => item.toJson()).toList(),
        'relationships': relationships.map((item) => item.toJson()).toList(),
      };
}

class Project {
  const Project({required this.id, required this.name, this.description = '', this.role, this.diagramCount = 0, this.accessCode});
  final String id;
  final String name;
  final String description;
  final String? role;
  final int diagramCount;
  final String? accessCode;

  factory Project.fromJson(Map<String, dynamic> json) => Project(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? 'Unnamed project',
        description: json['description'] as String? ?? '',
        role: json['role'] as String?,
        diagramCount: (json['diagramCount'] as num?)?.toInt() ?? 0,
        accessCode: json['accessCode'] as String?,
      );
}

class DiagramSummary {
  const DiagramSummary({required this.id, required this.name, this.updatedAt, this.version = 0, this.reviewNumber = 0});
  final String id;
  final String name;
  final String? updatedAt;
  final int version;
  final int reviewNumber;

  factory DiagramSummary.fromJson(Map<String, dynamic> json) => DiagramSummary(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? 'Untitled diagram',
        updatedAt: json['updatedAt'] as String?,
        version: (json['version'] as num?)?.toInt() ?? 0,
        reviewNumber: (json['reviewNumber'] as num?)?.toInt() ?? 0,
      );
}

class DiagramVersion {
  const DiagramVersion({required this.versionNumber, this.createdAt, this.createdBy = '', this.message, this.document, this.reviewNumber = 0});
  final int versionNumber;
  final String? createdAt;
  final String createdBy;
  final String? message;
  final UmlDocument? document;
  final int reviewNumber;

  factory DiagramVersion.fromJson(Map<String, dynamic> json) => DiagramVersion(
        versionNumber: (json['versionNumber'] as num?)?.toInt() ?? 0,
        createdAt: json['createdAt'] as String?,
        createdBy: json['createdBy'] as String? ?? '',
        message: json['message'] as String?,
        reviewNumber: (json['reviewNumber'] as num?)?.toInt() ?? 0,
        document: json['document'] is Map<String, dynamic>
            ? UmlDocument.fromJson(json['document'] as Map<String, dynamic>)
            : null,
      );
}
