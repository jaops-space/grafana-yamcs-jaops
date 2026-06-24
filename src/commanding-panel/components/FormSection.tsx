import React from 'react';

export function FormSection(props: {
  title?: string;
  description?: string;
  children: React.ReactNode;
  columns?: number;
  separated?: boolean;
}) {
  const { title, description, children, columns, separated = true } = props;

  return (
    <section
      style={{
        width: '100%',
        paddingTop: separated ? '10px' : 0,
        borderTop: separated ? '1px solid rgba(204, 204, 220, 0.16)' : undefined,
      }}
    >
      {(title || description) && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '2px', marginBottom: '8px' }}>
          {title && <h5 style={{ margin: 0, fontSize: '13px', fontWeight: 600 }}>{title}</h5>}
          {description && <span style={{ opacity: 0.75, fontSize: '11px', lineHeight: 1.3 }}>{description}</span>}
        </div>
      )}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: columns ? `repeat(${columns}, minmax(0, 1fr))` : 'repeat(auto-fit, minmax(180px, 1fr))',
          gap: '8px 12px',
          alignItems: 'start',
        }}
      >
        {children}
      </div>
    </section>
  );
}
