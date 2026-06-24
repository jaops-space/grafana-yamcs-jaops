import React from 'react';
import { SelectableValue } from '@grafana/data';
import { Combobox, Field } from '@grafana/ui';

export function CommandSelector(props: {
  label: string;
  description?: string;
  endpoint: string;
  datasource: any;
  value?: string | null;
  disabled?: boolean;
  commandInfo?: any;
  onChange: (name: string) => void;
}) {
  const { label, description, endpoint, datasource, value, disabled, commandInfo, onChange } = props;

  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'minmax(240px, 360px) 1fr', gap: '12px', alignItems: 'end' }}>
      <Field label={label} description={description} style={{ marginBottom: 0 }}>
        <Combobox
          disabled={disabled}
          value={value ?? null}
          isClearable
          options={async (q: string) => {
            if (!endpoint || !datasource) {
              return [];
            }

            const results: Array<{ name: string; description: string }> = await datasource.getResource(
              `endpoint/${endpoint}/commands`,
              q ? { q } : undefined
            );

            return results.map((c) => ({
              label: c.name,
              value: c.name,
              description: c.description,
            }));
          }}
          onChange={(e: SelectableValue<string> | null) => onChange(e?.value ?? '')}
        />
      </Field>

      <div style={{ minWidth: 0, paddingBottom: '2px' }}>
        <div style={{ fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {commandInfo?.name || commandInfo?.qualifiedName || 'No command selected'}
        </div>
        <div style={{ opacity: 0.75, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {commandInfo?.shortDescription || commandInfo?.longDescription || 'Select a command to configure its arguments.'}
        </div>
      </div>
    </div>
  );
}
