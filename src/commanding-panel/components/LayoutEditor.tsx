import React from 'react';
import { SelectableValue } from '@grafana/data';
import { Card, Combobox, Field, Input } from '@grafana/ui';
import { ButtonWidthMode, LayoutAlign, LayoutDirection, LayoutJustify } from '../utils/layout';

export function LayoutEditor(props: {
  options: any;
  onChange: (key: string, value: any) => void;
}) {
  const { options, onChange } = props;

  return (
    <Card style={{ margin: '8px', padding: '12px 14px' }}>
      <Card.Heading>
        <h4 style={{ margin: 0 }}>Runtime Button Layout</h4>
      </Card.Heading>
      <Card.Meta>Controls the runtime buttons and the group preview only. Edit cards keep their own layout.</Card.Meta>
      <Card.Description>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(170px, 1fr))',
            gap: '8px 12px',
            alignItems: 'end',
          }}
        >
          <Field label="Direction" style={{ marginBottom: 0 }}>
            <Combobox
              value={options.layoutDirection ?? 'column'}
              options={[
                { label: 'Vertical', value: 'column' },
                { label: 'Horizontal', value: 'row' },
              ]}
              onChange={(e: SelectableValue<LayoutDirection>) => onChange('layoutDirection', e.value)}
            />
          </Field>

          <Field label="Wrap" style={{ marginBottom: 0 }}>
            <Combobox
              value={String(options.layoutWrap ?? true)}
              options={[
                { label: 'Wrap', value: 'true' },
                { label: 'No wrap', value: 'false' },
              ]}
              onChange={(e: SelectableValue<string>) => onChange('layoutWrap', e.value === 'true')}
            />
          </Field>

          <Field label="Gap" description="px" style={{ marginBottom: 0 }}>
            <Input type="number" min={0} value={options.layoutGap ?? 4} onChange={(e) => onChange('layoutGap', Number(e.currentTarget.value))} />
          </Field>

          <Field label="Justify" style={{ marginBottom: 0 }}>
            <Combobox
              value={options.layoutJustify ?? 'flex-start'}
              options={[
                { label: 'Start', value: 'flex-start' },
                { label: 'Center', value: 'center' },
                { label: 'End', value: 'flex-end' },
                { label: 'Space between', value: 'space-between' },
                { label: 'Space around', value: 'space-around' },
                { label: 'Space evenly', value: 'space-evenly' },
              ]}
              onChange={(e: SelectableValue<LayoutJustify>) => onChange('layoutJustify', e.value)}
            />
          </Field>

          <Field label="Align" style={{ marginBottom: 0 }}>
            <Combobox
              value={options.layoutAlign ?? 'stretch'}
              options={[
                { label: 'Stretch', value: 'stretch' },
                { label: 'Start', value: 'flex-start' },
                { label: 'Center', value: 'center' },
                { label: 'End', value: 'flex-end' },
              ]}
              onChange={(e: SelectableValue<LayoutAlign>) => onChange('layoutAlign', e.value)}
            />
          </Field>

          <Field label="Width mode" style={{ marginBottom: 0 }}>
            <Combobox
              value={options.buttonWidthMode ?? 'fill'}
              options={[
                { label: 'Auto', value: 'auto' },
                { label: 'Equal / flexible', value: 'equal' },
                { label: 'Fixed', value: 'fixed' },
                { label: 'Fill row', value: 'fill' },
              ]}
              onChange={(e: SelectableValue<ButtonWidthMode>) => onChange('buttonWidthMode', e.value)}
            />
          </Field>

          <Field label="Min width" description="px" style={{ marginBottom: 0 }}>
            <Input type="number" min={0} value={options.buttonMinWidth ?? 160} onChange={(e) => onChange('buttonMinWidth', Number(e.currentTarget.value))} />
          </Field>

          <Field label="Width" description="px, for fixed/preferred" style={{ marginBottom: 0 }}>
            <Input type="number" min={0} value={options.buttonWidth ?? ''} onChange={(e) => onChange('buttonWidth', e.currentTarget.value === '' ? undefined : Number(e.currentTarget.value))} />
          </Field>

          <Field label="Min height" description="px" style={{ marginBottom: 0 }}>
            <Input type="number" min={0} value={options.buttonMinHeight ?? 40} onChange={(e) => onChange('buttonMinHeight', Number(e.currentTarget.value))} />
          </Field>

          <Field label="Height" description="px, optional" style={{ marginBottom: 0 }}>
            <Input type="number" min={0} value={options.buttonHeight ?? ''} onChange={(e) => onChange('buttonHeight', e.currentTarget.value === '' ? undefined : Number(e.currentTarget.value))} />
          </Field>
        </div>
      </Card.Description>
    </Card>
  );
}
