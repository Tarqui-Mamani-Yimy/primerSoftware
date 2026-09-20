import React, { useState } from 'react';
import { UMLClassNode, UMLAttribute, UMLMethod, Stereotype, UMLRelationship } from '../../types';
import { es } from '../../i18n/es';

interface InspectorProps {
  selectedClass: UMLClassNode;
  onUpdateClass: (updated: UMLClassNode) => void;
  onClose: () => void;
  onGenerateCode: () => void;
  onDeleteClass: (id: string) => void;
  classes: UMLClassNode[];
  relationships: UMLRelationship[];
}

export const Inspector: React.FC<InspectorProps> = ({
  selectedClass,
  onUpdateClass,
  onClose,
  onGenerateCode,
  onDeleteClass,
  classes,
  relationships
}) => {
  const [saveFeedback, setSaveFeedback] = useState(false);
  const [showNewAttrModal, setShowNewAttrModal] = useState(false);
  const [newAttrName, setNewAttrName] = useState('');
  const [newAttrType, setNewAttrType] = useState('String');
  const [newAttrVisibility, setNewAttrVisibility] = useState<'+' | '-' | '#'>('+');
  const connectedRelationshipCount = relationships.filter((relationship) => relationship.sourceId === selectedClass.id || relationship.targetId === selectedClass.id).length;

  const handleUpdateField = <K extends keyof UMLClassNode>(field: K, value: UMLClassNode[K]) => {
    onUpdateClass({
      ...selectedClass,
      [field]: value
    });
  };

  const handleAddAttribute = () => {
    if (!newAttrName.trim()) return;
    const newAttr: UMLAttribute = {
      id: `attr_${Date.now()}`,
      name: newAttrName.trim(),
      type: newAttrType,
      visibility: newAttrVisibility,
      annotations: newAttrType === 'UUID' ? ['@Id'] : ['@NotNull']
    };
    onUpdateClass({
      ...selectedClass,
      attributes: [...selectedClass.attributes, newAttr]
    });
    setNewAttrName('');
    setShowNewAttrModal(false);
  };

  const handleDeleteAttribute = (attrId: string) => {
    onUpdateClass({
      ...selectedClass,
      attributes: selectedClass.attributes.filter(a => a.id !== attrId)
    });
  };

  const handleAddMethod = () => {
    const methodName = prompt('Ingresá la firma del método (por ejemplo, recalcularDescuento(tasa))', 'recalcularDescuento()');
    if (!methodName) return;
    const newMethod: UMLMethod = {
      id: `meth_${Date.now()}`,
      name: methodName,
      returnType: 'BigDecimal',
      visibility: '+'
    };
    onUpdateClass({
      ...selectedClass,
      methods: [...selectedClass.methods, newMethod]
    });
  };

  const handleDeleteMethod = (methodId: string) => {
    onUpdateClass({
      ...selectedClass,
      methods: selectedClass.methods.filter(m => m.id !== methodId)
    });
  };

  const handleSave = () => {
    setSaveFeedback(true);
    setTimeout(() => setSaveFeedback(false), 1800);
  };

  const handleDeleteClass = () => {
    if (!window.confirm(es.canvas.deleteClassConfirm(selectedClass.name, connectedRelationshipCount))) return;
    onDeleteClass(selectedClass.id);
  };

  // An association class attaches to a relationship whose endpoints are both
  // ordinary classes (the association class itself can never be an endpoint).
  const eligibleRelationships = relationships.filter((relationship) => {
    if (relationship.sourceId === selectedClass.id || relationship.targetId === selectedClass.id) return false;
    const source = classes.find((c) => c.id === relationship.sourceId);
    const target = classes.find((c) => c.id === relationship.targetId);
    return Boolean(source && target && !source.isAssociationClass && !target.isAssociationClass);
  });

  const relationshipLabel = (relationship: UMLRelationship): string => {
    const source = classes.find((c) => c.id === relationship.sourceId);
    const target = classes.find((c) => c.id === relationship.targetId);
    return `${source?.name ?? relationship.sourceId} — ${target?.name ?? relationship.targetId} (${relationship.type})`;
  };

  return (
    <div className="w-80 flex-shrink-0 bg-[#181c24] flex flex-col z-20 shadow-xl border-l border-[#3c4a42] overflow-hidden">
      {/* Inspector Header */}
      <div className="p-3 bg-[#1c2028] border-b border-[#3c4a42] flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="material-symbols-outlined text-[#4edea3] text-sm">tune</span>
          <span className="font-heading text-sm uppercase tracking-wider text-[#dfe2ee] font-bold">Inspector</span>
        </div>
        <div className="flex items-center gap-2 font-mono">
          <span className="text-[#4edea3] text-[10px] font-bold">{selectedClass.name}.class</span>
          <button 
            onClick={onClose}
            className="text-[#bbcabf] hover:text-[#dfe2ee] transition-colors"
          >
            <span className="material-symbols-outlined text-sm">close</span>
          </button>
        </div>
      </div>

      {/* Properties Form Body */}
      <div className="flex-1 overflow-y-auto p-3 space-y-4 font-mono">
        {/* Class Metadata */}
        <div className="space-y-2 bg-[#1c2028] p-2.5 border border-[#3c4a42]">
          <div className="flex items-center justify-between pb-1 border-b border-[#3c4a42]">
            <span className="text-[10px] uppercase font-bold text-[#bbcabf]">Metadatos de clase</span>
            <span className="material-symbols-outlined text-xs text-[#4edea3]">data_object</span>
          </div>

          <div>
            <label className="text-[9px] uppercase text-[#bbcabf] block mb-1">Identificador de clase</label>
            <input
              className="w-full bg-[#0a0e16] px-2 py-1 text-[#dfe2ee] text-xs border border-[#3c4a42] focus:border-[#4edea3] focus:outline-none"
              type="text"
              value={selectedClass.name}
              onChange={(e) => handleUpdateField('name', e.target.value)}
            />
          </div>

          <div>
            <label className="text-[9px] uppercase text-[#bbcabf] block mb-1">Paquete destino</label>
            <input
              className="w-full bg-[#0a0e16] px-2 py-1 text-[#bbcabf] text-xs border border-[#3c4a42] focus:border-[#4edea3] focus:outline-none"
              type="text"
              value={selectedClass.package}
              onChange={(e) => handleUpdateField('package', e.target.value)}
            />
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-[9px] uppercase text-[#bbcabf] block mb-1">Estereotipo</label>
              <select
                className="w-full bg-[#0a0e16] px-2 py-1 text-[#4edea3] text-xs border border-[#3c4a42] focus:border-[#4edea3] focus:outline-none"
                value={selectedClass.stereotype}
                onChange={(e) => handleUpdateField('stereotype', e.target.value as Stereotype)}
              >
                <option value="«Entity»">«Entity»</option>
                <option value="«Entity, AggregateRoot»">«Entity, AggregateRoot»</option>
                <option value="«Entity, Part»">«Entity, Part»</option>
                <option value="«Abstract»">«Abstract»</option>
                <option value="«Interface»">«Interface»</option>
                <option value="«ValueObject»">«ValueObject»</option>
                <option value="«Repository»">«Repository»</option>
                <option value="«Service»">«Service»</option>
              </select>
            </div>
            <div>
              <label className="text-[9px] uppercase text-[#bbcabf] block mb-1">Tabla asociada</label>
              <input
                className="w-full bg-[#0a0e16] px-2 py-1 text-[#dfe2ee] text-xs border border-[#3c4a42] focus:border-[#4edea3] focus:outline-none"
                type="text"
                value={selectedClass.tableBinding}
                onChange={(e) => handleUpdateField('tableBinding', e.target.value)}
              />
            </div>

          </div>
        </div>

        {/* Association Class */}
        <div className="space-y-2 bg-[#1c2028] p-2.5 border border-[#3c4a42]">
          <div className="flex items-center justify-between pb-1 border-b border-[#3c4a42]">
            <span className="text-[10px] uppercase font-bold text-[#bbcabf]">{es.canvas.associationClass}</span>
            <span className="material-symbols-outlined text-xs text-[#c792ea]">alt_route</span>
          </div>

          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={Boolean(selectedClass.isAssociationClass)}
              onChange={(e) => handleUpdateField('isAssociationClass', e.target.checked)}
              className="accent-[#c792ea]"
            />
            <span className="text-xs text-[#dfe2ee]">{es.canvas.associationClassMark}</span>
          </label>

          {selectedClass.isAssociationClass && (
            <div>
              <label className="text-[9px] uppercase text-[#bbcabf] block mb-1">{es.canvas.associationClassAttached}</label>
              <select
                className="w-full bg-[#0a0e16] px-2 py-1 text-[#dfe2ee] text-xs border border-[#3c4a42] focus:border-[#c792ea] focus:outline-none"
                value={selectedClass.attachedRelationshipId ?? ''}
                onChange={(e) => handleUpdateField('attachedRelationshipId', e.target.value || undefined)}
              >
                <option value="">{es.canvas.associationClassNone}</option>
                {eligibleRelationships.map((relationship) => (
                  <option key={relationship.id} value={relationship.id}>
                    {relationshipLabel(relationship)}
                  </option>
                ))}
              </select>
              {eligibleRelationships.length === 0 && (
                <p className="mt-1 text-[10px] text-[#bbcabf]">{es.canvas.associationClassHint}</p>
              )}
            </div>
          )}
        </div>

        {/* Field / Attribute List Editor */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-[10px] uppercase text-[#dfe2ee] tracking-wider font-bold">
              Attributes ({selectedClass.attributes.length})
            </span>
            <button
              onClick={() => setShowNewAttrModal(true)}
              className="flex items-center gap-1 text-xs text-[#4edea3] hover:underline"
            >
              <span className="material-symbols-outlined text-xs">add</span>
              <span>Nuevo atributo</span>
            </button>
          </div>

          {showNewAttrModal && (
            <div className="p-2.5 bg-[#262a33] border border-[#4edea3] space-y-2">
              <div className="text-[10px] text-[#4edea3] font-bold uppercase">Agregar atributo a {selectedClass.name}</div>
              <div className="grid grid-cols-3 gap-1.5">
                <input
                  type="text"
                  placeholder="name"
                  value={newAttrName}
                  onChange={(e) => setNewAttrName(e.target.value)}
                  className="col-span-2 bg-[#0a0e16] px-1.5 py-1 text-xs border border-[#3c4a42] focus:outline-none text-[#dfe2ee]"
                />
                <select
                  value={newAttrType}
                  onChange={(e) => setNewAttrType(e.target.value)}
                  className="bg-[#0a0e16] px-1 py-1 text-xs border border-[#3c4a42] text-[#4cd7f6]"
                >
                  <option value="String">String</option>
                  <option value="BigDecimal">BigDecimal</option>
                  <option value="Integer">Integer</option>
                  <option value="UUID">UUID</option>
                  <option value="Instant">Instant</option>
                  <option value="Boolean">Boolean</option>
                </select>
              </div>
              <div className="flex items-center justify-between pt-1">
                <div className="flex items-center gap-1 text-[10px]">
                  <span className="text-[#bbcabf]">Vis:</span>
                  {(['+', '-', '#'] as const).map(vis => (
                    <button
                      key={vis}
                      onClick={() => setNewAttrVisibility(vis)}
                      className={`px-1.5 py-0.5 border text-xs ${newAttrVisibility === vis ? 'bg-[#4edea3] text-black font-bold' : 'bg-[#1c2028] text-white border-[#3c4a42]'}`}
                    >
                      {vis}
                    </button>
                  ))}
                </div>
                <div className="flex items-center gap-1.5">
                  <button
                    onClick={() => setShowNewAttrModal(false)}
                    className="px-2 py-0.5 text-xs text-[#bbcabf] hover:text-white"
                  >
                    Cancelar
                  </button>
                  <button
                    onClick={handleAddAttribute}
                    className="px-2 py-0.5 bg-[#4edea3] text-black text-xs font-bold"
                  >
                    Agregar
                  </button>
                </div>
              </div>
            </div>
          )}

          <div className="space-y-1.5">
            {selectedClass.attributes.map((attr) => (
              <div key={attr.id} className="p-2 bg-[#1c2028] border border-[#3c4a42] space-y-1">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-1.5">
                    <span className="text-[#4cd7f6] font-bold text-xs">{attr.visibility}</span>
                    <span className="text-[#dfe2ee] font-bold text-xs">{attr.name}</span>
                  </div>
                  <span className="text-[#4cd7f6] text-[11px]">{attr.type}</span>
                </div>

                <div className="flex items-center justify-between text-[10px] text-[#bbcabf] pt-1 border-t border-[#3c4a42]">
                  <span className="text-[#4edea3]">
                    {attr.annotations && attr.annotations.length > 0 ? attr.annotations.join(' ') : 'none'}
                  </span>
                  <button
                    onClick={() => handleDeleteAttribute(attr.id)}
                    className="text-[#bbcabf] hover:text-[#ffb4ab] transition-colors"
                    title="Eliminar atributo"
                  >
                    <span className="material-symbols-outlined text-xs">delete</span>
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Operations / Method Manager */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-[10px] uppercase text-[#dfe2ee] tracking-wider font-bold">
              Operations ({selectedClass.methods.length})
            </span>
            <button
              onClick={handleAddMethod}
              className="flex items-center gap-1 text-xs text-[#4edea3] hover:underline"
            >
              <span className="material-symbols-outlined text-xs">add</span>
              <span>Nuevo método</span>
            </button>
          </div>

          <div className="space-y-1 text-xs">
            {selectedClass.methods.map((method) => (
              <div key={method.id} className="flex items-center justify-between p-2 bg-[#1c2028] border border-[#3c4a42]">
                <div className="truncate mr-2">
                  <span className="text-[#4edea3] font-bold">{method.visibility} </span>
                  <span className="text-[#dfe2ee] truncate">{method.name}</span>
                </div>
                <div className="flex items-center gap-1.5 flex-shrink-0">
                  <span className="text-[#4cd7f6] text-[11px]">{method.returnType}</span>
                  <button
                    onClick={() => handleDeleteMethod(method.id)}
                    className="text-[#86948a] hover:text-[#ffb4ab]"
                    title="Eliminar método"
                  >
                    <span className="material-symbols-outlined text-xs">close</span>
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Target Code Preview Card */}
        <div className="p-2.5 bg-[#31353e] space-y-2 border border-[#3c4a42]">
          <span className="text-[9px] uppercase text-[#bbcabf] block font-bold">Vista previa del código destino</span>
          <div className="bg-[#0a0e16] p-2 text-[10px] text-[#dfe2ee] space-y-0.5 border border-[#3c4a42]">
            <span className="text-[#4edea3]">@Entity</span><br />
            <span className="text-[#4edea3]">@Table</span>(name = <span className="text-[#4cd7f6]">"{selectedClass.tableBinding}"</span>)<br />
            <span className="text-[#dfe2ee] font-bold">public class {selectedClass.name} &#123;</span><br />
            &nbsp;&nbsp;<span className="text-[#d0bcff]">@Id</span> UUID id;<br />
            &nbsp;&nbsp;<span className="text-[#86948a]">// +{selectedClass.attributes.length - 1} attributes &amp; {selectedClass.methods.length} methods</span><br />
            &#125;
          </div>

          <button
            onClick={onGenerateCode}
            className="w-full py-1.5 bg-[#1c2028] hover:bg-[#262a33] text-[#4edea3] border border-[#4edea3] text-xs uppercase text-center font-bold transition-colors flex items-center justify-center gap-1.5 shadow-sm"
          >
            <span className="material-symbols-outlined text-xs">bolt</span>
            <span>Generar DDL Spring + repositorio</span>
          </button>
        </div>
      </div>

      {/* Inspector Footer */}
      <div className="border-t border-[#3c4a42] bg-[#1c2028] p-3">
        <button
          type="button"
          onClick={handleDeleteClass}
          aria-label={`${es.canvas.deleteClass}: ${selectedClass.name}`}
          className="w-full border border-[#ffb4ab] px-3 py-2 text-xs font-bold uppercase text-[#ffb4ab] transition-colors hover:bg-[#3a2529] focus:outline-none focus:ring-2 focus:ring-[#ffb4ab]"
        >
          {es.canvas.deleteClass}
        </button>
      </div>
      <div className="p-2.5 bg-[#1c2028] border-t border-[#3c4a42] flex items-center justify-between font-mono">
        <span className="text-xs text-[#bbcabf]">
          Sincronización del esquema: <strong className="text-[#4edea3]">Automática</strong>
        </span>
        <button
          onClick={handleSave}
          className="px-3 py-1 bg-[#4edea3] text-black text-xs uppercase font-bold hover:brightness-110 active:scale-95 transition-all flex items-center gap-1"
        >
          {saveFeedback ? (
            <>
              <span className="material-symbols-outlined text-xs">done</span>
              <span>¡Guardado!</span>
            </>
          ) : (
            <span>Guardar entidad</span>
          )}
        </button>
      </div>
    </div>
  );
};
