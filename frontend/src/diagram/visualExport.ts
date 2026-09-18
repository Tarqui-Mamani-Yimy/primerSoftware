import { RelationshipType, UMLClassNode, UMLRelationship } from '../types';

const PADDING = 48;
const MIN_SIZE = 240;
const MAX_RASTER_DIMENSION = 8192;

const nodeWidth = (umlClass: UMLClassNode) => umlClass.width ?? 250;
const nodeHeight = (umlClass: UMLClassNode) => Math.max(130, 92 + (umlClass.attributes.length + umlClass.methods.length) * 18);
const dashByRelationship: Partial<Record<RelationshipType, string>> = { realization: '6 4', dependency: '6 4' };

const escapeXml = (value: string) => value.replace(/[<>&'\"]/g, (character) => ({
  '<': '&lt;',
  '>': '&gt;',
  '&': '&amp;',
  "'": '&apos;',
  '"': '&quot;'
}[character] ?? character));

const relationshipColor = (type: RelationshipType) => (
  type === 'composition' ? '#4edea3' : type === 'generalization' || type === 'realization' ? '#d0bcff' : '#4cd7f6'
);

const markerEnd = (type: RelationshipType) => {
  if (type === 'aggregation' || type === 'composition') return '';
  if (type === 'generalization' || type === 'realization') return ' marker-end="url(#triangle)"';
  return ' marker-end="url(#arrow)"';
};

const markerStart = (type: RelationshipType) => (
  type === 'aggregation' ? ' marker-start="url(#aggregation)"' : type === 'composition' ? ' marker-start="url(#composition)"' : ''
);

const textLine = (x: number, y: number, content: string, options = '') => `<text x="${x}" y="${y}" ${options}>${escapeXml(content)}</text>`;

export interface DiagramSvgExport {
  filename: string;
  height: number;
  svg: string;
  width: number;
}

/** Creates a self-contained SVG of the complete UML model, independent of the viewport. */
export const createDiagramSvgExport = (diagramName: string, classes: UMLClassNode[], relationships: UMLRelationship[]): DiagramSvgExport => {
  const left = classes.length ? Math.min(...classes.map((item) => item.x)) - PADDING : 0;
  const top = classes.length ? Math.min(...classes.map((item) => item.y)) - PADDING : 0;
  const right = classes.length ? Math.max(...classes.map((item) => item.x + nodeWidth(item))) + PADDING : MIN_SIZE;
  const bottom = classes.length ? Math.max(...classes.map((item) => item.y + nodeHeight(item))) + PADDING : MIN_SIZE;
  const width = Math.max(MIN_SIZE, right - left);
  const height = Math.max(MIN_SIZE, bottom - top);
  const validRelationships = relationships.flatMap((relationship) => {
    const source = classes.find((item) => item.id === relationship.sourceId);
    const target = classes.find((item) => item.id === relationship.targetId);
    return source && target ? [{ relationship, source, target }] : [];
  });

  const relationshipMarkup = validRelationships.map(({ relationship, source, target }) => {
    const color = relationshipColor(relationship.type);
    const x1 = source.x + nodeWidth(source) / 2;
    const y1 = source.y + nodeHeight(source) / 2;
    const x2 = target.x + nodeWidth(target) / 2;
    const y2 = target.y + nodeHeight(target) / 2;
    const dash = dashByRelationship[relationship.type] ? ` stroke-dasharray="${dashByRelationship[relationship.type]}"` : '';
    const labels = [
      relationship.sourceMultiplicity && textLine(x1 + 8, y1 - 8, relationship.sourceMultiplicity, `fill="${color}" font-size="12"`),
      relationship.targetMultiplicity && textLine(x2 + 8, y2 - 8, relationship.targetMultiplicity, `fill="${color}" font-size="12"`),
      relationship.label && textLine((x1 + x2) / 2, (y1 + y2) / 2 - 8, relationship.label, `fill="${color}" font-size="12" text-anchor="middle"`)
    ].filter(Boolean).join('');
    return `<g><line x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}" stroke="${color}" stroke-width="2"${dash}${markerStart(relationship.type)}${markerEnd(relationship.type)}/>${labels}</g>`;
  }).join('');

  const classMarkup = classes.map((umlClass) => {
    const width = nodeWidth(umlClass);
    const height = nodeHeight(umlClass);
    const attributeStart = umlClass.y + 65;
    const methodStart = umlClass.y + 81 + umlClass.attributes.length * 18;
    const attributes = umlClass.attributes.map((attribute, index) => textLine(umlClass.x + 12, attributeStart + index * 18, `${attribute.visibility} ${attribute.name}: ${attribute.type}`, 'fill="#dfe2ee" font-size="11"')).join('');
    const methods = umlClass.methods.map((method, index) => textLine(umlClass.x + 12, methodStart + index * 18, `${method.visibility} ${method.name}: ${method.returnType}`, 'fill="#dfe2ee" font-size="11"')).join('');
    const attributeDivider = umlClass.y + 64 + umlClass.attributes.length * 18;
    return `<g><rect x="${umlClass.x}" y="${umlClass.y}" width="${width}" height="${height}" rx="2" fill="#1c2028" stroke="#3c4a42" stroke-width="1.5"/><line x1="${umlClass.x}" y1="${umlClass.y + 48}" x2="${umlClass.x + width}" y2="${umlClass.y + 48}" stroke="#3c4a42"/><line x1="${umlClass.x}" y1="${attributeDivider}" x2="${umlClass.x + width}" y2="${attributeDivider}" stroke="#3c4a42"/>${textLine(umlClass.x + width / 2, umlClass.y + 20, umlClass.stereotype, 'fill="#4edea3" font-size="11" text-anchor="middle"')}${textLine(umlClass.x + width / 2, umlClass.y + 38, umlClass.name, 'fill="#4edea3" font-size="14" font-weight="bold" text-anchor="middle"')}${attributes}${methods}</g>`;
  }).join('');

  const safeFilename = (diagramName.trim() || 'uml-diagram').replace(/[^a-z0-9_-]+/gi, '-').replace(/^-+|-+$/g, '') || 'uml-diagram';
  return {
    filename: safeFilename.toLowerCase(),
    width,
    height,
    svg: `<?xml version="1.0" encoding="UTF-8"?><svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="${left} ${top} ${width} ${height}" role="img" aria-label="${escapeXml(diagramName || 'UML diagram')}"><defs><marker id="arrow" markerWidth="10" markerHeight="10" refX="8" refY="5" orient="auto"><path d="M 0 0 L 10 5 L 0 10 z" fill="#4cd7f6"/></marker><marker id="triangle" markerWidth="12" markerHeight="12" refX="10" refY="6" orient="auto"><path d="M 0 0 L 12 6 L 0 12 z" fill="#1c2028" stroke="#d0bcff" stroke-width="1.5"/></marker><marker id="aggregation" markerWidth="13" markerHeight="13" refX="2" refY="6" orient="auto"><path d="M 7 0 L 13 6 L 7 12 L 1 6 z" fill="#1c2028" stroke="#bbcabf" stroke-width="1.5"/></marker><marker id="composition" markerWidth="13" markerHeight="13" refX="2" refY="6" orient="auto"><path d="M 7 0 L 13 6 L 7 12 L 1 6 z" fill="#4edea3" stroke="#4edea3" stroke-width="1.5"/></marker></defs><rect x="${left}" y="${top}" width="${width}" height="${height}" fill="#0a0e16"/>${relationshipMarkup}${classMarkup}</svg>`
  };
};

const downloadBlob = (blob: Blob, filename: string) => {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 0);
};

export const downloadDiagramSvg = (diagramName: string, classes: UMLClassNode[], relationships: UMLRelationship[]) => {
  const exportData = createDiagramSvgExport(diagramName, classes, relationships);
  downloadBlob(new Blob([exportData.svg], { type: 'image/svg+xml;charset=utf-8' }), `${exportData.filename}.svg`);
};

export const downloadDiagramPng = async (diagramName: string, classes: UMLClassNode[], relationships: UMLRelationship[]) => {
  const exportData = createDiagramSvgExport(diagramName, classes, relationships);
  const scale = Math.min(2, MAX_RASTER_DIMENSION / Math.max(exportData.width, exportData.height));
  const image = new Image();
  const svgUrl = URL.createObjectURL(new Blob([exportData.svg], { type: 'image/svg+xml;charset=utf-8' }));
  try {
    await new Promise<void>((resolve, reject) => {
      image.onload = () => resolve();
      image.onerror = () => reject(new Error('The diagram could not be rasterized.'));
      image.src = svgUrl;
    });
    const canvas = document.createElement('canvas');
    canvas.width = Math.max(1, Math.round(exportData.width * scale));
    canvas.height = Math.max(1, Math.round(exportData.height * scale));
    const context = canvas.getContext('2d');
    if (!context) throw new Error('Canvas export is not available in this browser.');
    context.drawImage(image, 0, 0, canvas.width, canvas.height);
    const png = await new Promise<Blob>((resolve, reject) => canvas.toBlob((blob) => blob ? resolve(blob) : reject(new Error('The PNG could not be created.')), 'image/png'));
    downloadBlob(png, `${exportData.filename}.png`);
  } finally {
    URL.revokeObjectURL(svgUrl);
  }
};
