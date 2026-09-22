import { RelationshipType } from '../types';
import { MULTIPLICITY_PATTERN } from './relationshipHelpers';

export type VoiceCommand =
  | { kind: 'create-class'; name: string }
  | {
      kind: 'create-relationship';
      sourceName: string;
      targetName: string;
      type?: RelationshipType;
      sourceMultiplicity?: string;
      targetMultiplicity?: string;
      label?: string;
    }
  | { kind: 'undo-voice-command' }
  | { kind: 'add-attribute'; name: string; type: string; className: string }
  | { kind: 'add-method'; name: string; returnType: string; className: string };

const CLASS_PATTERN = /^crear\s+(?:una\s+)?clase\s+(.+)$/i;
const RELATIONSHIP_START_PATTERN = /^relacionar\s+(.+?)\s+con\s+(.+)$/i;
const UNDO_PATTERN = /^deshacer(?:\s+(?:comando\s+de\s+voz|último\s+comando\s+de\s+voz))?$/i;
const ATTRIBUTE_PATTERN = /^(?:agregar|añadir)\s+atributo\s+(.+?)\s+de\s+tipo\s+(.+?)\s+a\s+(.+)$/i;
const METHOD_PATTERN = /^(?:agregar|añadir)\s+método\s+(.+?)\s+de\s+retorno\s+(.+?)\s+a\s+(.+)$/i;
const NAME_PATTERN = /^[\p{L}\p{N}][\p{L}\p{N}' -]*$/u;

function cleanName(value: string): string | null {
  const name = value.trim().replace(/\s+/g, ' ');
  return name.length > 0 && name.length <= 80 && NAME_PATTERN.test(name) ? name : null;
}

function cleanLabel(value: string): string | null {
  const label = value.trim().replace(/\s+/g, ' ');
  return label.length > 0 && label.length <= 80 && NAME_PATTERN.test(label) ? label : null;
}

function parseRelationshipType(value: string): RelationshipType | null {
  const norm = value.trim().toLowerCase();
  if (/^(?:asociaci[oó]n|association)$/.test(norm)) return 'association';
  if (/^(?:agregaci[oó]n|aggregation)$/.test(norm)) return 'aggregation';
  if (/^(?:composici[oó]n|composition)$/.test(norm)) return 'composition';
  if (/^(?:generalizaci[oó]n|generalization)$/.test(norm)) return 'generalization';
  if (/^(?:realizaci[oó]n|realization)$/.test(norm)) return 'realization';
  if (/^(?:dependencia|dependency)$/.test(norm)) return 'dependency';
  return null;
}

// SPOKEN_NUMBERS maps the small Spanish cardinal vocabulary a speech-to-text
// transcript can produce for a UML multiplicity bound to its digit form.
const SPOKEN_NUMBERS: Record<string, string> = {
  cero: '0',
  uno: '1',
  dos: '2',
  tres: '3',
  cuatro: '4',
  cinco: '5',
  seis: '6',
  siete: '7',
  ocho: '8',
  nueve: '9',
  diez: '10',
};

/** Resolves one spoken or literal multiplicity bound token ("uno", "5", "muchos", "*") to its digit/"*" form, or null when unrecognized. */
function spokenBound(token: string): string | null {
  const norm = token.trim().toLowerCase();
  if (norm === '*' || norm === 'muchos' || norm === 'varios' || norm === 'asterisco' || norm === 'estrella' || norm === 'n') {
    return '*';
  }
  if (/^\d+$/.test(norm)) return norm;
  return SPOKEN_NUMBERS[norm] ?? null;
}

/**
 * Parses one multiplicity payload into MULTIPLICITY_PATTERN form. Accepts the
 * literal digit/"*" forms plus the spoken Spanish vocabulary a Deepgram
 * transcript produces: a single value ("uno", "muchos"), "X a Y", "de X a Y",
 * "X punto punto Y", the literal "X..Y", "X o más"/"X o mas" ("X..*"), and the
 * idiom "cero o uno" ("0..1").
 */
function cleanMultiplicity(value: string): string | null {
  const trimmed = value.trim().replace(/\s*\.\.\s*/g, '..');
  if (MULTIPLICITY_PATTERN.test(trimmed)) return trimmed;

  const lower = trimmed.toLowerCase();

  if (/^cero\s+o\s+uno$/.test(lower)) return '0..1';

  const orMore = /^(.+?)\s+o\s+m[aá]s$/.exec(lower);
  if (orMore) {
    const lo = spokenBound(orMore[1]);
    return lo ? checkRange(`${lo}..*`) : null;
  }

  const deXaY = /^de\s+(.+?)\s+a\s+(.+)$/.exec(lower);
  if (deXaY) {
    const lo = spokenBound(deXaY[1]);
    const hi = spokenBound(deXaY[2]);
    return lo && hi ? checkRange(`${lo}..${hi}`) : null;
  }

  const puntoPunto = /^(.+?)\s+punto\s+punto\s+(.+)$/.exec(lower);
  if (puntoPunto) {
    const lo = spokenBound(puntoPunto[1]);
    const hi = spokenBound(puntoPunto[2]);
    return lo && hi ? checkRange(`${lo}..${hi}`) : null;
  }

  const xaY = /^(.+?)\s+a\s+(.+)$/.exec(lower);
  if (xaY) {
    const lo = spokenBound(xaY[1]);
    const hi = spokenBound(xaY[2]);
    return lo && hi ? checkRange(`${lo}..${hi}`) : null;
  }

  const single = spokenBound(lower);
  return single ? checkRange(single) : null;
}

function checkRange(value: string): string | null {
  return MULTIPLICITY_PATTERN.test(value) ? value : null;
}

function parseCardinalities(payload: string): { sourceMultiplicity?: string; targetMultiplicity?: string } | null {
  const norm = payload.trim();
  const bothSrcDst = /^origen\s+(.+?)\s+y\s+(?:cardinalidad\s+)?destino\s+(.+)$/i.exec(norm);
  if (bothSrcDst) {
    const src = cleanMultiplicity(bothSrcDst[1]);
    const dst = cleanMultiplicity(bothSrcDst[2]);
    return src && dst ? { sourceMultiplicity: src, targetMultiplicity: dst } : null;
  }
  const bothDstSrc = /^destino\s+(.+?)\s+y\s+(?:cardinalidad\s+)?origen\s+(.+)$/i.exec(norm);
  if (bothDstSrc) {
    const dst = cleanMultiplicity(bothDstSrc[1]);
    const src = cleanMultiplicity(bothDstSrc[2]);
    return src && dst ? { sourceMultiplicity: src, targetMultiplicity: dst } : null;
  }
  const srcOnly = /^origen\s+(.+)$/i.exec(norm);
  if (srcOnly) {
    const src = cleanMultiplicity(srcOnly[1]);
    return src ? { sourceMultiplicity: src } : null;
  }
  const dstOnly = /^destino\s+(.+)$/i.exec(norm);
  if (dstOnly) {
    const dst = cleanMultiplicity(dstOnly[1]);
    return dst ? { targetMultiplicity: dst } : null;
  }
  return null;
}

function parseRelationshipCommand(text: string): VoiceCommand | null {
  const match = RELATIONSHIP_START_PATTERN.exec(text);
  if (!match) return null;

  const sourceName = cleanName(match[1]);
  if (!sourceName) return null;

  const rest = match[2].trim();
  const markerRegex = /\b(como|con\s+cardinalidad|con\s+verbo)\b/gi;
  const markers: Array<{ marker: string; index: number; length: number }> = [];
  let m: RegExpExecArray | null;
  while ((m = markerRegex.exec(rest)) !== null) {
    markers.push({
      marker: m[1].toLowerCase().replace(/\s+/g, ' '),
      index: m.index,
      length: m[0].length,
    });
  }

  let targetRaw: string;
  const modifierTokens: Array<{ marker: string; payload: string }> = [];

  if (markers.length === 0) {
    targetRaw = rest;
  } else {
    targetRaw = rest.slice(0, markers[0].index);
    for (let i = 0; i < markers.length; i++) {
      const current = markers[i];
      const start = current.index + current.length;
      const end = i + 1 < markers.length ? markers[i + 1].index : rest.length;
      const payload = rest.slice(start, end).trim();
      modifierTokens.push({ marker: current.marker, payload });
    }
  }

  const targetName = cleanName(targetRaw);
  if (!targetName) return null;

  let relationshipType: RelationshipType | undefined;
  let sourceMultiplicity: string | undefined;
  let targetMultiplicity: string | undefined;
  let label: string | undefined;

  let hasType = false;
  let hasLabel = false;

  for (const token of modifierTokens) {
    if (token.marker === 'como') {
      if (hasType) return null;
      hasType = true;
      const parsedType = parseRelationshipType(token.payload);
      if (!parsedType) return null;
      relationshipType = parsedType;
    } else if (token.marker === 'con cardinalidad') {
      const cards = parseCardinalities(token.payload);
      if (!cards) return null;
      if (cards.sourceMultiplicity !== undefined) {
        if (sourceMultiplicity !== undefined) return null;
        sourceMultiplicity = cards.sourceMultiplicity;
      }
      if (cards.targetMultiplicity !== undefined) {
        if (targetMultiplicity !== undefined) return null;
        targetMultiplicity = cards.targetMultiplicity;
      }
    } else if (token.marker === 'con verbo') {
      if (hasLabel) return null;
      hasLabel = true;
      const cleanedLabel = cleanLabel(token.payload);
      if (!cleanedLabel) return null;
      label = cleanedLabel;
    } else {
      return null;
    }
  }

  return {
    kind: 'create-relationship',
    sourceName,
    targetName,
    ...(relationshipType !== undefined ? { type: relationshipType } : {}),
    ...(sourceMultiplicity !== undefined ? { sourceMultiplicity } : {}),
    ...(targetMultiplicity !== undefined ? { targetMultiplicity } : {}),
    ...(label !== undefined ? { label } : {}),
  };
}

// INNER_PUNCTUATION_PATTERN matches the punctuation Deepgram's smart_format
// inserts inside a transcript (commas, semicolons, colons, Spanish inverted
// marks, and straight/curly/angle quotes) that would otherwise break
// downstream patterns such as cleanName.
const INNER_PUNCTUATION_PATTERN = /[,;:¿¡"“”«»]/g;

/** Strips smart_format punctuation and collapses whitespace before parsing. */
function normalizeTranscript(raw: string): string {
  const withoutPunctuation = raw.replace(INNER_PUNCTUATION_PATTERN, ' ');
  const collapsed = withoutPunctuation.replace(/\s+/g, ' ').trim();
  return collapsed.replace(/[.!?]+$/g, '').trim();
}

/**
 * Maps the small Spanish attribute/return type vocabulary (case/accent
 * insensitive) a voice transcript can produce to the canonical Java type
 * name jdlgen expects. Unknown types (including already-canonical English
 * names such as "String") pass through unchanged.
 */
function mapSpanishType(value: string): string {
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

/** Parses only the small, confirmed Spanish voice vocabulary. */
export function parseVoiceCommand(transcript: string): VoiceCommand | null {
  const text = normalizeTranscript(transcript);
  // Parameters are intentionally outside this first member-command vocabulary.
  if (/\bpar[aá]metros?\b/i.test(text)) return null;
  if (UNDO_PATTERN.test(text)) return { kind: 'undo-voice-command' };

  const attributeMatch = ATTRIBUTE_PATTERN.exec(text);
  if (attributeMatch) {
    const name = cleanName(attributeMatch[1]);
    const type = cleanName(attributeMatch[2]);
    const className = cleanName(attributeMatch[3]);
    return name && type && className
      ? { kind: 'add-attribute', name, type: mapSpanishType(type), className }
      : null;
  }

  const methodMatch = METHOD_PATTERN.exec(text);
  if (methodMatch) {
    const name = cleanName(methodMatch[1]);
    const returnType = cleanName(methodMatch[2]);
    const className = cleanName(methodMatch[3]);
    return name && returnType && className
      ? { kind: 'add-method', name, returnType: mapSpanishType(returnType), className }
      : null;
  }

  const classMatch = CLASS_PATTERN.exec(text);
  if (classMatch) {
    const name = cleanName(classMatch[1]);
    return name ? { kind: 'create-class', name } : null;
  }

  return parseRelationshipCommand(text);
}
