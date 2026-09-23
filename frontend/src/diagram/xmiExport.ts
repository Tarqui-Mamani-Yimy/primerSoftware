import { UMLClassNode, UMLRelationship, Visibility } from '../types';

// UML VisibilityKind literals. The app stores visibility as '+', '-', '#'
// (and '~' for package, if it is ever introduced); unknown values are
// omitted from the XMI rather than emitted verbatim.
const visibilityKind = (value: Visibility): string | undefined => ({
  '+': 'public',
  '-': 'private',
  '#': 'protected',
  '~': 'package'
} as Record<string, string>)[value];

const escapeXml = (value: string) => value.replace(/[<>&'\"]/g, (character) => ({
  '<': '&lt;',
  '>': '&gt;',
  '&': '&amp;',
  "'": '&apos;',
  '"': '&quot;'
}[character] ?? character));

// XMI ids are XML IDs, so user supplied ids must not be used verbatim. The
// deterministic suffix also keeps ids stable between exports of the same model.
const hash = (value: string) => {
  let result = 2166136261;
  for (let index = 0; index < value.length; index += 1) {
    result ^= value.charCodeAt(index);
    result = Math.imul(result, 16777619);
  }
  return (result >>> 0).toString(36);
};

const xmiId = (kind: string, source: string) => {
  const readable = source.trim().replace(/[^A-Za-z0-9_.-]+/g, '_').replace(/^[^A-Za-z_]+/, 'id_') || 'item';
  return `${kind}_${readable}_${hash(source)}`;
};

const safeFilename = (name: string) => (
  name.trim().replace(/[^a-z0-9_-]+/gi, '-').replace(/^-+|-+$/g, '').toLowerCase() || 'uml-diagram'
);

const classifierKind = (umlClass: UMLClassNode) => (
  umlClass.stereotype === '«Interface»' ? 'uml:Interface' : umlClass.stereotype === '«Enum»' ? 'uml:Enumeration' : 'uml:Class'
);

const multiplicity = (value?: string) => {
  if (!value?.trim()) return '';
  const match = value.trim().match(/^(\d+|\*)\s*(?:\.\.\s*(\d+|\*))?$/);
  if (!match) return '';
  const lower = match[2] ? match[1] : match[1] === '*' ? '0' : match[1];
  const upper = match[2] ?? match[1];
  if (lower === '*' || (upper !== '*' && Number(lower) > Number(upper))) return '';
  return `<lowerValue xmi:type="uml:LiteralInteger" value="${lower}"/><upperValue xmi:type="uml:LiteralUnlimitedNatural" value="${upper}"/>`;
};

const endpoint = (id: string, typeId: string, multiplicityValue: string | undefined, aggregation?: 'shared' | 'composite') => (
  `<ownedEnd xmi:type="uml:Property" xmi:id="${id}" type="${typeId}"${aggregation ? ` aggregation="${aggregation}"` : ''}>${multiplicity(multiplicityValue)}</ownedEnd>`
);

export interface DiagramXmiExport {
  filename: string;
  xmi: string;
}

/** Creates UML 2.1 XMI using only standard UML metaclasses and XMI references. */
export const createDiagramXmiExport = (diagramName: string, classes: UMLClassNode[], relationships: UMLRelationship[]): DiagramXmiExport => {
  const modelId = xmiId('model', diagramName || 'uml-model');
  const classIds = new Map(classes.map((umlClass) => [umlClass.id, xmiId('classifier', umlClass.id)]));
  const classesById = new Map(classes.map((umlClass) => [umlClass.id, umlClass]));
  const classifierByName = new Map(classes.map((umlClass) => [umlClass.name.trim(), classIds.get(umlClass.id)!]));
  const customTypes = new Map<string, string>();
  const resolveType = (type: string) => {
    const value = type.trim();
    if (!value || value.toLowerCase() === 'void') return undefined;
    const classifierId = classifierByName.get(value);
    if (classifierId) return classifierId;
    if (!customTypes.has(value)) customTypes.set(value, xmiId('datatype', value));
    return customTypes.get(value)!;
  };

  // Resolve member types first so local UML DataTypes can be emitted before use.
  classes.forEach((umlClass) => {
    umlClass.attributes.forEach((attribute) => resolveType(attribute.type));
    umlClass.methods.forEach((method) => resolveType(method.returnType));
  });

  const dataTypes = [...customTypes.entries()].map(([name, id]) => (
    `<packagedElement xmi:type="uml:DataType" xmi:id="${id}" name="${escapeXml(name)}"/>`
  )).join('');

  const relationshipElements = relationships.flatMap((relationship) => {
    const source = classIds.get(relationship.sourceId);
    const target = classIds.get(relationship.targetId);
    if (!source || !target) return [];
    const relationshipId = xmiId('relationship', relationship.id || `${relationship.sourceId}:${relationship.targetId}:${relationship.type}`);
    const label = relationship.label?.trim() ? ` name="${escapeXml(relationship.label.trim())}"` : '';
    if (relationship.type === 'generalization') {
      return [`<generalization xmi:id="${relationshipId}" general="${target}"${label}/>`];
    }
    // Interface realizations are owned by their implementing classifier below.
    if (relationship.type === 'realization') return [];
    if (relationship.type === 'dependency') {
      return [`<packagedElement xmi:type="uml:Dependency" xmi:id="${relationshipId}" client="${source}" supplier="${target}"${label}/>`];
    }
    // The canvas draws the diamond at the source end (the whole), but per UML
    // notation `aggregation` is set on the member end typed by the PART —
    // i.e. the end opposite the whole, which is the target here.
    const aggregation = relationship.type === 'aggregation' ? 'shared' : relationship.type === 'composition' ? 'composite' : undefined;
    const sourceEnd = xmiId('associationEnd', `${relationshipId}:source`);
    const targetEnd = xmiId('associationEnd', `${relationshipId}:target`);
    return [`<packagedElement xmi:type="uml:Association" xmi:id="${relationshipId}"${label} memberEnd="${sourceEnd} ${targetEnd}">${endpoint(sourceEnd, source, relationship.sourceMultiplicity)}${endpoint(targetEnd, target, relationship.targetMultiplicity, aggregation)}</packagedElement>`];
  });

  // Generalizations are owned by the specific classifier in UML, rather than by the model.
  const generalizationsBySource = new Map<string, string[]>();
  const realizationsBySource = new Map<string, string[]>();
  relationships.forEach((relationship) => {
    if (relationship.type !== 'generalization' && relationship.type !== 'realization') return;
    const source = classIds.get(relationship.sourceId);
    const target = classIds.get(relationship.targetId);
    if (!source || !target) return;
    const sourceClass = classesById.get(relationship.sourceId);
    const targetClass = classesById.get(relationship.targetId);
    if (relationship.type === 'realization' && (!sourceClass || !targetClass || sourceClass.stereotype === '«Interface»' || sourceClass.stereotype === '«Enum»' || targetClass.stereotype !== '«Interface»')) return;
    const id = xmiId('relationship', relationship.id || `${relationship.sourceId}:${relationship.targetId}:${relationship.type}`);
    const label = relationship.label?.trim() ? ` name="${escapeXml(relationship.label.trim())}"` : '';
    if (relationship.type === 'generalization') {
      generalizationsBySource.set(relationship.sourceId, [...(generalizationsBySource.get(relationship.sourceId) ?? []), `<generalization xmi:id="${id}" general="${target}"${label}/>`]);
    } else {
      realizationsBySource.set(relationship.sourceId, [...(realizationsBySource.get(relationship.sourceId) ?? []), `<interfaceRealization xmi:id="${id}" client="${source}" supplier="${target}" contract="${target}"${label}/>`]);
    }
  });
  const classifiersWithGeneralizations = classes.map((umlClass) => {
    const id = classIds.get(umlClass.id)!;
    const isEnumeration = umlClass.stereotype === '«Enum»';
    const attributes = isEnumeration ? umlClass.attributes.map((attribute, index) => (
      `<ownedLiteral xmi:type="uml:EnumerationLiteral" xmi:id="${xmiId('literal', `${umlClass.id}:${attribute.id || index}`)}" name="${escapeXml(attribute.name)}"/>`
    )).join('') : umlClass.attributes.map((attribute, index) => {
      const type = resolveType(attribute.type);
      const visibility = visibilityKind(attribute.visibility);
      return `<ownedAttribute xmi:type="uml:Property" xmi:id="${xmiId('attribute', `${umlClass.id}:${attribute.id || index}`)}" name="${escapeXml(attribute.name)}"${visibility ? ` visibility="${visibility}"` : ''}${type ? ` type="${type}"` : ''}/>`;
    }).join('');
    const methods = umlClass.methods.map((method, index) => {
      const type = resolveType(method.returnType);
      const visibility = visibilityKind(method.visibility);
      return `<ownedOperation xmi:type="uml:Operation" xmi:id="${xmiId('operation', `${umlClass.id}:${method.id || index}`)}" name="${escapeXml(method.name)}"${visibility ? ` visibility="${visibility}"` : ''}${method.isAbstract ? ' isAbstract="true"' : ''}>${type ? `<ownedParameter xmi:type="uml:Parameter" xmi:id="${xmiId('return', `${umlClass.id}:${method.id || index}`)}" direction="return" type="${type}"/>` : ''}</ownedOperation>`;
    }).join('');
    const abstract = umlClass.isAbstract || umlClass.stereotype === '«Abstract»' ? ' isAbstract="true"' : '';
    return `<packagedElement xmi:type="${classifierKind(umlClass)}" xmi:id="${id}" name="${escapeXml(umlClass.name)}"${abstract}>${attributes}${methods}${(generalizationsBySource.get(umlClass.id) ?? []).join('')}${(realizationsBySource.get(umlClass.id) ?? []).join('')}</packagedElement>`;
  }).join('');

  // Generalizations and realizations are owned by their source classifier.
  const modelRelationships = relationshipElements.filter((element) => !element.startsWith('<generalization'));
  return {
    filename: safeFilename(diagramName),
    xmi: `<?xml version="1.0" encoding="UTF-8"?>\n<xmi:XMI xmlns:xmi="http://schema.omg.org/spec/XMI/2.1" xmlns:uml="http://schema.omg.org/spec/UML/2.1" xmi:version="2.1">\n  <uml:Model xmi:id="${modelId}" name="${escapeXml(diagramName.trim() || 'UML Model')}">\n    ${dataTypes}${classifiersWithGeneralizations}${modelRelationships.join('')}\n  </uml:Model>\n</xmi:XMI>\n`
  };
};

export const downloadDiagramXmi = (diagramName: string, classes: UMLClassNode[], relationships: UMLRelationship[]) => {
  const exported = createDiagramXmiExport(diagramName, classes, relationships);
  const url = URL.createObjectURL(new Blob([exported.xmi], { type: 'application/vnd.omg.xmi+xml;charset=utf-8' }));
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = `${exported.filename}.xmi`;
  anchor.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 0);
};
