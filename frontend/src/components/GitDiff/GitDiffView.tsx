import React, { useState } from 'react';

export const GitDiffView: React.FC = () => {
  const [selectedCommit, setSelectedCommit] = useState('#7f3a9e');
  const [viewMode, setViewMode] = useState<'split' | 'unified'>('split');

  const diffLines = [
    { type: 'info', text: '@@ -12,18 +12,28 @@ public class Order {' },
    { type: 'context', text: '     @Id @GeneratedValue(strategy = GenerationType.UUID)' },
    { type: 'context', text: '     private UUID id;' },
    { type: 'del', text: '-    // payment_id old column removed in favor of polymorphism' },
    { type: 'del', text: '-    private String legacyPaymentStatus;' },
    { type: 'add', text: '+    @NotNull' },
    { type: 'add', text: '+    @PositiveOrZero' },
    { type: 'add', text: '+    @Column(name = "total_amount", precision = 12, scale = 2, nullable = false)' },
    { type: 'add', text: '+    private BigDecimal totalAmount;' },
    { type: 'add', text: '+    @ManyToOne(fetch = FetchType.LAZY)' },
    { type: 'add', text: '+    private Customer customer;' },
    { type: 'context', text: '     @Column(name = "created_at", nullable = false, updatable = false)' },
    { type: 'context', text: '     private Instant createdAt = Instant.now();' }
  ];

  return (
    <div className="flex flex-col w-full min-h-[calc(100vh-4rem)] bg-[#0f131c] font-mono">
      {/* Header bar */}
      <div className="p-3 bg-[#181c24] border-b border-[#3c4a42] flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <span className="material-symbols-outlined text-[#d0bcff]">history_toggle_off</span>
          <div>
            <h2 className="font-heading text-sm text-[#dfe2ee] font-bold">Branch &amp; Schema Diff</h2>
            <p className="text-[11px] text-[#bbcabf]">Comparing feat/payment-order-v2 (HEAD) against origin/main</p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <div className="flex items-center bg-[#1c2028] border border-[#3c4a42] p-0.5 text-xs">
            <button
              onClick={() => setViewMode('unified')}
              className={`px-2 py-1 ${viewMode === 'unified' ? 'bg-[#262a33] text-[#4edea3] font-bold' : 'text-[#bbcabf]'}`}
            >
              Unified
            </button>
            <button
              onClick={() => setViewMode('split')}
              className={`px-2 py-1 ${viewMode === 'split' ? 'bg-[#262a33] text-[#4edea3] font-bold' : 'text-[#bbcabf]'}`}
            >
              Split View
            </button>
          </div>

          <button className="px-3 py-1.5 bg-[#4edea3] text-black font-bold text-xs uppercase flex items-center gap-1.5 shadow-md">
            <span className="material-symbols-outlined text-xs">merge_type</span>
            <span>Create Pull Request</span>
          </button>
        </div>
      </div>

      <div className="grid grid-cols-12 flex-1">
        {/* Commits list */}
        <div className="col-span-12 md:col-span-3 bg-[#181c24] border-r border-[#3c4a42] p-3 space-y-2">
          <div className="text-[10px] uppercase font-bold text-[#bbcabf] mb-2">Schema Commits on Branch</div>
          
          {[
            { hash: '#7f3a9e', msg: 'feat: add polymorphic payment & order JPA mappings', author: 'You', time: '14 min ago' },
            { hash: '#d48a12', msg: 'refactor: switch Order status to EnumType.STRING', author: 'Sarah Chen', time: '1 hr ago' },
            { hash: '#b190f4', msg: 'schema: init customer and order_item migrations', author: 'Alex Kumar', time: '3 hrs ago' }
          ].map(c => (
            <div
              key={c.hash}
              onClick={() => setSelectedCommit(c.hash)}
              className={`p-2 border cursor-pointer transition-colors text-xs ${
                selectedCommit === c.hash
                  ? 'bg-[#1c2028] border-[#4edea3] text-[#dfe2ee]'
                  : 'bg-[#0a0e16] border-[#3c4a42] text-[#bbcabf] hover:border-[#86948a]'
              }`}
            >
              <div className="flex items-center justify-between text-[10px] text-[#4cd7f6] mb-1">
                <span className="font-bold">{c.hash}</span>
                <span className="text-[#86948a]">{c.time}</span>
              </div>
              <p className="line-clamp-2 text-white font-semibold">{c.msg}</p>
              <span className="text-[10px] text-[#86948a] block mt-1">by {c.author}</span>
            </div>
          ))}
        </div>

        {/* Diff content view */}
        <div className="col-span-12 md:col-span-9 bg-[#0a0e16] p-4 overflow-x-auto text-xs">
          <div className="flex items-center justify-between pb-2 mb-3 border-b border-[#3c4a42]">
            <div className="flex items-center gap-2">
              <span className="text-[#4edea3] font-bold">src/main/java/com/architect/domain/model/Order.java</span>
              <span className="px-1.5 py-0.5 bg-[#10b981]/20 text-[#4edea3] text-[10px] font-bold">+6 -2</span>
            </div>
            <span className="text-[10px] text-[#86948a]">Diff generated with Git 2.44</span>
          </div>

          <div className="bg-[#181c24] border border-[#3c4a42] p-3 space-y-1">
            {diffLines.map((line, idx) => (
              <div
                key={idx}
                className={`py-0.5 px-2 flex items-start gap-2 ${
                  line.type === 'add'
                    ? 'bg-[#10b981]/15 text-[#4edea3]'
                    : line.type === 'del'
                    ? 'bg-[#93000a]/30 text-[#ffb4ab]'
                    : line.type === 'info'
                    ? 'text-[#4cd7f6] bg-[#00424e]/20'
                    : 'text-[#bbcabf]'
                }`}
              >
                <span className="w-4 text-center select-none font-bold">
                  {line.type === 'add' ? '+' : line.type === 'del' ? '-' : ' '}
                </span>
                <span className="whitespace-pre">{line.text}</span>
              </div>
            ))}
          </div>

          {/* DDL Schema Diff Preview */}
          <div className="mt-6 flex items-center justify-between pb-2 mb-3 border-b border-[#3c4a42]">
            <div className="flex items-center gap-2">
              <span className="text-[#d0bcff] font-bold">src/main/resources/db/migration/V1__init_schema.sql</span>
              <span className="px-1.5 py-0.5 bg-[#10b981]/20 text-[#4edea3] text-[10px] font-bold">+18 lines</span>
            </div>
          </div>

          <div className="bg-[#181c24] border border-[#3c4a42] p-3 space-y-1 text-[#dfe2ee]">
            <div className="text-[#4edea3] bg-[#10b981]/15 py-0.5 px-2">+ CREATE TABLE t_payments_cc (payment_id UUID PRIMARY KEY REFERENCES t_payments(id));</div>
            <div className="text-[#4edea3] bg-[#10b981]/15 py-0.5 px-2">+ CREATE TABLE t_payments_stripe (payment_id UUID PRIMARY KEY REFERENCES t_payments(id));</div>
            <div className="text-[#4edea3] bg-[#10b981]/15 py-0.5 px-2">+ CREATE INDEX idx_orders_customer_id ON t_orders(customer_id);</div>
          </div>
        </div>
      </div>
    </div>
  );
};
